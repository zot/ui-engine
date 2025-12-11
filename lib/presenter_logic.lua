-- CRC: crc-LuaPresenterLogic.md
-- Spec: interfaces.md
-- Presenter logic utilities for Lua presenters

local PresenterLogic = {}

-- Define a new presenter type with methods
function PresenterLogic.defineType(name, definition)
    local pt = {
        _type = name,
        _methods = {},
        _properties = {}
    }

    -- Copy methods from definition
    for k, v in pairs(definition) do
        if type(v) == "function" then
            pt._methods[k] = v
            pt[k] = v
        else
            pt[k] = v
        end
    end

    -- Set metatable for inheritance
    setmetatable(pt, {
        __index = function(t, key)
            return rawget(t, key)
        end
    })

    -- Register with runtime
    ui.registerPresenter(name, pt)

    return pt
end

-- Create an instance of a presenter type
function PresenterLogic.instantiate(typeName, props)
    -- This would be called from Go to create instances
    local pt = _G[typeName] or {}
    local instance = setmetatable({}, {__index = pt})

    -- Set properties
    if props then
        for k, v in pairs(props) do
            instance[k] = v
        end
    end

    -- Call init if exists
    if instance.init then
        instance:init()
    end

    return instance
end

-- Handle an action on a presenter instance
function PresenterLogic.handleAction(instance, action, values)
    -- Parse method call syntax: "methodName()"
    local methodName = action:match("^(%w+)%(%)$") or action

    local method = instance[methodName]
    if method and type(method) == "function" then
        local result, err = pcall(method, instance, values)
        if not result then
            return nil, "Action failed: " .. tostring(err)
        end
        return err, nil  -- pcall returns (true, result) on success
    end

    return nil, "Unknown action: " .. action
end

-- Update a property on a presenter instance
function PresenterLogic.updateProperty(instance, key, value)
    instance[key] = value

    -- Check for setter
    local props = rawget(instance, "_properties")
    if props and props[key] and props[key].set then
        props[key].set(instance, value)
    end
end

-- Get a property from a presenter instance
function PresenterLogic.getProperty(instance, key)
    -- Check for getter
    local props = rawget(instance, "_properties")
    if props and props[key] and props[key].get then
        return props[key].get(instance)
    end

    return instance[key]
end

-- Notify that presenter state changed (triggers variable update)
function PresenterLogic.notifyChange(instance)
    if instance.notifyChange then
        instance:notifyChange()
    end
end

return PresenterLogic
