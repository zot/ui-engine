-- CRC: crc-Presenter.md
-- Spec: main.md
-- Base presenter type for Lua presenters

local Presenter = {}
Presenter.__index = Presenter

-- Create a new presenter type
function Presenter.defineType(name)
    local pt = setmetatable({}, Presenter)
    pt._type = name
    pt._methods = {}
    pt._properties = {}

    -- Register with runtime
    ui.registerPresenter(name, pt)

    return pt
end

-- Add a method to presenter type
function Presenter:defineMethod(name, fn)
    self._methods[name] = fn
    self[name] = fn
end

-- Add a property with getter/setter
function Presenter:defineProperty(name, getter, setter)
    self._properties[name] = {
        get = getter,
        set = setter
    }
end

-- Initialize a presenter instance (override in subclasses)
function Presenter:init()
    -- Default implementation does nothing
end

-- Get presenter type name
function Presenter:getType()
    return self._type
end

-- Convert presenter to variable data
function Presenter:toData()
    local data = {}
    for k, v in pairs(self) do
        if type(v) ~= "function" and not k:match("^_") then
            data[k] = v
        end
    end
    return data
end

-- Handle action from frontend
function Presenter:handleAction(action, values)
    local method = self[action]
    if method and type(method) == "function" then
        return method(self, values)
    end
    return nil, "Unknown action: " .. action
end

-- Notify that presenter state changed
function Presenter:notifyChange()
    -- This would be implemented to trigger variable update
    ui.log("Presenter state changed: " .. tostring(self._type))
end

return Presenter
