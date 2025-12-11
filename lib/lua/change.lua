-- CRC: crc-ChangeDetector.md
-- Spec: libraries.md
-- Lua change detection library

local ChangeDetector = {}
ChangeDetector.__index = ChangeDetector

-- Create a new change detector
function ChangeDetector.new(connection, navigator)
    local self = setmetatable({}, ChangeDetector)
    self.conn = connection
    self.navigator = navigator
    self.watchedVariables = {} -- varId -> path
    self.previousValues = {}   -- varId -> last known value
    self.pendingRefresh = false
    self.throttleInterval = 0.05 -- 50ms
    self.lastRefresh = 0
    self.rootObject = nil
    return self
end

-- Set the root object for path resolution
function ChangeDetector:setRootObject(root)
    self.rootObject = root
end

-- Set throttle interval (in seconds)
function ChangeDetector:setThrottleInterval(interval)
    self.throttleInterval = interval
end

-- Add a watch on a variable
function ChangeDetector:addWatch(varId, path)
    self.watchedVariables[varId] = path

    -- Capture initial value
    if self.rootObject then
        local val, _ = self.navigator:resolve(self.rootObject, path)
        if val ~= nil then
            self.previousValues[varId] = self:cloneValue(val)
        end
    end
end

-- Remove a watch
function ChangeDetector:removeWatch(varId)
    self.watchedVariables[varId] = nil
    self.previousValues[varId] = nil
end

-- Check if a variable is watched
function ChangeDetector:isWatched(varId)
    return self.watchedVariables[varId] ~= nil
end

-- Refresh all watched variables and detect changes
function ChangeDetector:refresh()
    if not self.rootObject then
        return 0
    end

    local changes = {}

    for varId, path in pairs(self.watchedVariables) do
        local currentVal, err = self.navigator:resolve(self.rootObject, path)
        if not err then
            local prevVal = self.previousValues[varId]
            if not self:valuesEqual(prevVal, currentVal) then
                changes[varId] = currentVal
            end
        end
    end

    -- Send updates and update previous values
    local changeCount = 0
    for varId, val in pairs(changes) do
        self:sendUpdate(varId, val)
        self.previousValues[varId] = self:cloneValue(val)
        changeCount = changeCount + 1
    end

    self.lastRefresh = os.clock()
    self.pendingRefresh = false

    return changeCount
end

-- Schedule a refresh with throttling
function ChangeDetector:scheduleRefresh()
    if self.pendingRefresh then
        return
    end

    local elapsed = os.clock() - self.lastRefresh
    if elapsed < self.throttleInterval then
        self.pendingRefresh = true
        -- In real implementation, would schedule a timer
        return
    end

    self:refresh()
end

-- Trigger refresh after message receipt
function ChangeDetector:afterMessage()
    self:scheduleRefresh()
end

-- Send an update message
function ChangeDetector:sendUpdate(varId, value)
    if not self.conn or not self.conn:isConnected() then
        return
    end
    self.conn:update(varId, value, nil)
end

-- Compare two values for equality
function ChangeDetector:valuesEqual(a, b)
    if a == nil and b == nil then
        return true
    end
    if a == nil or b == nil then
        return false
    end

    local typeA = type(a)
    local typeB = type(b)

    if typeA ~= typeB then
        return false
    end

    if typeA ~= "table" then
        return a == b
    end

    -- Table comparison
    for k, v in pairs(a) do
        if not self:valuesEqual(v, b[k]) then
            return false
        end
    end

    for k, v in pairs(b) do
        if a[k] == nil then
            return false
        end
    end

    return true
end

-- Clone a value for comparison
function ChangeDetector:cloneValue(v)
    if v == nil then
        return nil
    end

    local t = type(v)
    if t ~= "table" then
        return v
    end

    local clone = {}
    for k, val in pairs(v) do
        clone[k] = self:cloneValue(val)
    end
    return clone
end

-- Get number of watched variables
function ChangeDetector:watchedCount()
    local count = 0
    for _ in pairs(self.watchedVariables) do
        count = count + 1
    end
    return count
end

-- Clear all watches and reset state
function ChangeDetector:clear()
    self.watchedVariables = {}
    self.previousValues = {}
    self.pendingRefresh = false
end

return ChangeDetector
