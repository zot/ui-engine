-- CRC: crc-LuaSession.md, crc-LuaVariable.md
-- Spec: libraries.md
-- Sequence: seq-lua-action-dispatch.md
-- Common Lua session API for embedded and external backends

local json = require("dkjson") or { encode = function() return "{}" end, decode = function() return {} end }

--------------------------------------------------
-- Variable wrapper class
--------------------------------------------------

local Variable = {}
Variable.__index = Variable

-- Create a new Variable wrapper (internal use only)
function Variable._new(session, id)
    local self = setmetatable({}, Variable)
    self._session = session
    self._id = id
    return self
end

-- Get variable ID
function Variable:getId()
    return self._id
end

-- Get current value
function Variable:getValue()
    return self._session:_getVariableValue(self._id)
end

-- Get a property value
function Variable:getProperty(name)
    return self._session:_getVariableProperty(self._id, name)
end

-- Update value (and optionally properties)
function Variable:update(value, properties)
    self._session:_updateVariable(self._id, value, properties)
end

-- Update only properties (not value)
function Variable:updateProperties(properties)
    self._session:_updateVariable(self._id, nil, properties)
end

--------------------------------------------------
-- Session class
--------------------------------------------------

local Session = {}
Session.__index = Session

-- Backend types
Session.BACKEND_EMBEDDED = "embedded"
Session.BACKEND_EXTERNAL = "external"

-- Create a new session
-- For external backends: pass connection object
-- For embedded backends: backend functions are injected by Go runtime
function Session.new(backend)
    local self = setmetatable({}, Session)

    self._backend = backend  -- Connection for external, nil for embedded (Go injects functions)
    self._backendType = backend and Session.BACKEND_EXTERNAL or Session.BACKEND_EMBEDDED
    self._variables = {}     -- varId -> Variable wrapper cache
    self._watchers = {}      -- varId -> { property -> callbacks[] }
    self._actionHandlers = {}-- actionName -> handler function

    -- Weak-keyed table: object -> varId (for consistent object references)
    self._objectToId = setmetatable({}, { __mode = "k" })

    return self
end

-- Get the app variable (variable 1)
function Session:getAppVariable()
    return self:getVariable(1)
end

-- Get a variable by ID (cached)
function Session:getVariable(id)
    if self._variables[id] then
        return self._variables[id]
    end

    local var = Variable._new(self, id)
    self._variables[id] = var
    return var
end

-- Create a child variable with value and properties
-- Returns the new Variable wrapper
function Session:createVariable(parentId, value, properties)
    local id = self:_createVariable(parentId, value, properties)
    if not id then
        return nil
    end

    -- Track object -> id mapping if value is a table
    if type(value) == "table" then
        self._objectToId[value] = id
    end

    return self:getVariable(id)
end

-- Destroy a variable by ID
function Session:destroyVariable(id)
    self:_destroyVariable(id)

    -- Remove from cache
    self._variables[id] = nil

    -- Remove watchers
    self._watchers[id] = nil
end

-- Watch a variable for any changes (value or properties)
function Session:watchVariable(id, callback)
    return self:watchProperty(id, "*", callback)
end

-- Watch a specific property on a variable
function Session:watchProperty(id, property, callback)
    if not self._watchers[id] then
        self._watchers[id] = {}
    end
    if not self._watchers[id][property] then
        self._watchers[id][property] = {}
    end

    table.insert(self._watchers[id][property], callback)
end

-- Register an action handler
function Session:registerActionHandler(name, handler)
    self._actionHandlers[name] = handler
end

-- Dispatch an action (called when action property changes)
function Session:dispatchAction(varId, actionName)
    local handler = self._actionHandlers[actionName]
    if not handler then
        print("[session] No handler for action: " .. tostring(actionName))
        return false
    end

    local var = self:getVariable(varId)
    local argsJson = var:getProperty("action-args") or "[]"
    local args = json.decode(argsJson) or {}

    -- Call handler with session, variable, and args
    local ok, err = pcall(handler, self, var, table.unpack(args))
    if not ok then
        print("[session] Action handler error: " .. tostring(err))
        return false
    end

    return true
end

-- Track an object -> variable ID mapping (for consistent object references)
function Session:trackObject(obj, varId)
    if type(obj) == "table" then
        self._objectToId[obj] = varId
    end
end

-- Get variable ID for a tracked object (nil if not tracked)
function Session:getObjectId(obj)
    return self._objectToId[obj]
end

-- Internal: notify property watchers (called by backend when variable updates)
function Session:_notifyPropertyChange(varId, property, value)
    local varWatchers = self._watchers[varId]
    if not varWatchers then
        return
    end

    -- Call property-specific watchers
    local propCallbacks = varWatchers[property]
    if propCallbacks then
        for _, cb in ipairs(propCallbacks) do
            pcall(cb, value)
        end
    end

    -- Call wildcard watchers
    local wildcardCallbacks = varWatchers["*"]
    if wildcardCallbacks then
        for _, cb in ipairs(wildcardCallbacks) do
            pcall(cb, value, property)
        end
    end

    -- Special handling for action property
    if property == "action" and value and value ~= "" then
        self:dispatchAction(varId, value)
    end
end

--------------------------------------------------
-- Backend operations (to be implemented by backend)
--------------------------------------------------

-- Get variable value (implemented by backend)
function Session:_getVariableValue(id)
    if self._backendType == Session.BACKEND_EXTERNAL then
        -- External: use connection
        local resp = self._backend:get(id)
        if resp and resp.result and resp.result[1] then
            return resp.result[1].value
        end
        return nil
    else
        -- Embedded: Go runtime injects this
        if self._getValueFn then
            return self._getValueFn(id)
        end
        return nil
    end
end

-- Get variable property (implemented by backend)
function Session:_getVariableProperty(id, name)
    if self._backendType == Session.BACKEND_EXTERNAL then
        local resp = self._backend:get(id)
        if resp and resp.result and resp.result[1] then
            local props = resp.result[1].properties
            return props and props[name]
        end
        return nil
    else
        -- Embedded: Go runtime injects this
        if self._getPropertyFn then
            return self._getPropertyFn(id, name)
        end
        return nil
    end
end

-- Create variable (implemented by backend)
function Session:_createVariable(parentId, value, properties)
    if self._backendType == Session.BACKEND_EXTERNAL then
        return self._backend:create(parentId, value, properties)
    else
        -- Embedded: Go runtime injects this
        if self._createFn then
            return self._createFn(parentId, value, properties)
        end
        return nil
    end
end

-- Update variable (implemented by backend)
function Session:_updateVariable(id, value, properties)
    if self._backendType == Session.BACKEND_EXTERNAL then
        return self._backend:update(id, value, properties)
    else
        -- Embedded: Go runtime injects this
        if self._updateFn then
            return self._updateFn(id, value, properties)
        end
    end
end

-- Destroy variable (implemented by backend)
function Session:_destroyVariable(id)
    if self._backendType == Session.BACKEND_EXTERNAL then
        return self._backend:destroy(id)
    else
        -- Embedded: Go runtime injects this
        if self._destroyFn then
            return self._destroyFn(id)
        end
    end
end

--------------------------------------------------
-- Embedded backend injection points
-- These are called by Go runtime to inject native functions
--------------------------------------------------

function Session:_setGetValueFn(fn)
    self._getValueFn = fn
end

function Session:_setGetPropertyFn(fn)
    self._getPropertyFn = fn
end

function Session:_setCreateFn(fn)
    self._createFn = fn
end

function Session:_setUpdateFn(fn)
    self._updateFn = fn
end

function Session:_setDestroyFn(fn)
    self._destroyFn = fn
end

return {
    Session = Session,
    Variable = Variable
}
