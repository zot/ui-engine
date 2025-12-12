-- Example custom wrapper implementation in Lua
-- CRC: crc-Wrapper.md
-- Spec: protocol.md
-- Sequence: seq-wrapper-transform.md
--
-- This file demonstrates how to create custom wrappers in Lua.
-- Wrappers transform variable values before they're sent to the frontend.
--
-- Example use case: A "Count" wrapper that transforms arrays into
-- objects with count information.
--
-- Usage in HTML viewdef:
--   <div ui-value="items?wrapper=CountDisplay">

-- CountDisplay wrapper - transforms array to {count: N, items: [...]}
local CountDisplay = {}
CountDisplay.__index = CountDisplay

-- computeValue transforms the raw value
-- @param self The wrapper instance
-- @param rawValue The raw value from path resolution (Lua table)
-- @return The transformed value to send to frontend (Lua table)
function CountDisplay:computeValue(rawValue)
    if type(rawValue) ~= "table" then
        return { count = 0, items = {} }
    end

    -- Check if it's an array
    local isArray = true
    local count = 0
    for k, _ in pairs(rawValue) do
        if type(k) ~= "number" then
            isArray = false
            break
        end
        count = count + 1
    end

    if not isArray then
        return { count = 0, items = {} }
    end

    return {
        count = count,
        items = rawValue,
        hasItems = count > 0,
        isEmpty = count == 0
    }
end

-- destroy is called when the variable is destroyed (optional)
function CountDisplay:destroy()
    -- No cleanup needed for this simple wrapper
end

-- Register the wrapper type
-- Note: This is commented out by default as it's just an example.
-- Uncomment to use.
-- ui.registerWrapper("CountDisplay", CountDisplay)

return CountDisplay
