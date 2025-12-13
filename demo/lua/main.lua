-- Simple Adder Demo
-- Two inputs that add to produce a result

local Adder = {type = "Adder"}
Adder.__index = Adder

function Adder:new()
    local tbl = {
        value1 = "",
        value2 = ""
    }
    setmetatable(tbl, self)
    return tbl
end

-- compute() is called via ui-value="compute()" binding
-- Returns the sum if both values are numbers, empty string otherwise
function Adder:compute()
    local n1 = tonumber(self.value1)
    local n2 = tonumber(self.value2)
    if n1 and n2 then
        return tostring(n1 + n2)
    end
    return ""
end

local app = Adder:new()
session:createAppVariable(app)

ui.log("Adder initialized")
