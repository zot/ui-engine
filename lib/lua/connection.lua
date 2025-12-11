-- CRC: crc-BackendConnection.md
-- Spec: libraries.md
-- Sequence: seq-lua-action-dispatch.md
-- Lua backend connection library
-- This is the transport layer for external Lua backends.
-- Use session.lua's Session class with a Connection for the full API.

local Connection = {}
Connection.__index = Connection

-- Create a new connection
function Connection.new(socketPath)
    local self = setmetatable({}, Connection)
    self.socketPath = socketPath or "/tmp/ui.sock"
    self.sessionId = nil
    self.rootVariableId = 1
    self.connected = false
    self.messageQueue = {}
    self.onCloseCallback = nil
    self.propertyWatchers = {} -- varId -> { property -> callbacks[] }
    -- Note: Actual socket operations would need luasocket or similar
    return self
end

-- Connect to UI server
function Connection:connect()
    -- This would use luasocket to connect
    -- For now, mark as connected
    self.connected = true
    return true
end

-- Disconnect from UI server
function Connection:disconnect()
    if not self.connected then
        return
    end

    self.connected = false
    if self.onCloseCallback then
        self.onCloseCallback()
    end
end

-- Check if connected
function Connection:isConnected()
    return self.connected
end

-- Register close callback
function Connection:onClose(fn)
    self.onCloseCallback = fn
end

-- Send a message (returns response)
function Connection:send(msg)
    if not self.connected then
        return nil, "not connected"
    end

    -- This would encode and send via socket
    -- For now, return mock response
    table.insert(self.messageQueue, msg)
    return { result = {} }
end

-- Set root value on variable 1
function Connection:setRootValue(value)
    return self:send({
        type = "update",
        id = 1,
        value = value
    })
end

-- Create a new variable
function Connection:create(parentId, value, properties)
    local resp, err = self:send({
        type = "create",
        parentId = parentId,
        value = value,
        properties = properties
    })
    if err then
        return nil, err
    end
    if resp and resp.result and resp.result.id then
        return resp.result.id
    end
    return nil, "unexpected response"
end

-- Update a variable
function Connection:update(id, value, properties)
    return self:send({
        type = "update",
        id = id,
        value = value,
        properties = properties
    })
end

-- Destroy a variable
function Connection:destroy(id)
    return self:send({
        type = "destroy",
        id = id
    })
end

-- Watch a variable
function Connection:watch(id)
    return self:send({
        type = "watch",
        id = id
    })
end

-- Unwatch a variable
function Connection:unwatch(id)
    return self:send({
        type = "unwatch",
        id = id
    })
end

-- Get variable values
function Connection:get(...)
    local ids = {...}
    return self:send({
        type = "get",
        varIds = ids
    })
end

-- Poll for pending messages
function Connection:poll(wait)
    return self:send({
        type = "poll",
        wait = wait
    })
end

-- Get session ID
function Connection:getSessionId()
    return self.sessionId
end

-- Set session ID
function Connection:setSessionId(id)
    self.sessionId = id
end

-- Attempt reconnection
function Connection:reconnect()
    if self.connected then
        self:disconnect()
    end
    return self:connect()
end

-- Register a property watcher (called by session to handle updates from server)
function Connection:watchProperty(varId, property, callback)
    if not self.propertyWatchers[varId] then
        self.propertyWatchers[varId] = {}
    end
    if not self.propertyWatchers[varId][property] then
        self.propertyWatchers[varId][property] = {}
    end
    table.insert(self.propertyWatchers[varId][property], callback)
end

-- Notify property watchers (called when server sends update)
function Connection:notifyPropertyChange(varId, property, value)
    local varWatchers = self.propertyWatchers[varId]
    if not varWatchers then
        return
    end

    local propCallbacks = varWatchers[property]
    if propCallbacks then
        for _, cb in ipairs(propCallbacks) do
            pcall(cb, value)
        end
    end

    -- Also notify wildcard watchers
    local wildcardCallbacks = varWatchers["*"]
    if wildcardCallbacks then
        for _, cb in ipairs(wildcardCallbacks) do
            pcall(cb, value, property)
        end
    end
end

-- Create a session using this connection
-- Convenience method to create a Session with this Connection as backend
function Connection:createSession()
    local sessionModule = require("session")
    return sessionModule.Session.new(self)
end

return Connection
