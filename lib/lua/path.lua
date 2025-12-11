-- CRC: crc-PathNavigator.md
-- Spec: protocol.md, libraries.md
-- Lua path navigation library

local PathNavigator = {}
PathNavigator.__index = PathNavigator

-- Create a new path navigator
function PathNavigator.new()
    local self = setmetatable({}, PathNavigator)
    self.pathCache = {}
    self.standardVars = {}
    return self
end

-- Register a standard @name variable
function PathNavigator:registerStandardVariable(name, value)
    self.standardVars[name] = value
end

-- Parse a path string into segments
function PathNavigator:parsePath(path)
    if self.pathCache[path] then
        return self.pathCache[path]
    end

    local segments = {}
    local parts = {}

    -- Split on dots
    for part in path:gmatch("[^.]+") do
        table.insert(parts, part)
    end

    for i, part in ipairs(parts) do
        local seg = { value = part }

        -- Check for @name at start
        if i == 1 and part:match("^@") then
            seg.type = "standard"
            seg.value = part:sub(2)
            table.insert(segments, seg)

        -- Check for parent traversal
        elseif part == ".." then
            seg.type = "parent"
            table.insert(segments, seg)

        -- Check for method call
        elseif part:match("^(%w+)%(%)$") then
            seg.type = "method"
            seg.value = part:match("^(%w+)%(%)$")
            table.insert(segments, seg)

        -- Check for array index (1-based)
        elseif part:match("^[1-9]%d*$") then
            seg.type = "index"
            seg.index = tonumber(part)
            table.insert(segments, seg)

        -- Default: property access
        else
            seg.type = "property"
            table.insert(segments, seg)
        end
    end

    self.pathCache[path] = segments
    return segments
end

-- Navigate a path to get a value
function PathNavigator:resolve(root, path)
    local segments = self:parsePath(path)
    if #segments == 0 then
        return root, nil
    end

    local current = root

    for _, seg in ipairs(segments) do
        if current == nil then
            return nil, "cannot navigate nil value at " .. tostring(seg.value)
        end

        local newVal, err = self:navigateSegment(current, seg)
        if err then
            return nil, err
        end
        current = newVal
    end

    return current, nil
end

-- Handle a single path segment
function PathNavigator:navigateSegment(current, seg)
    if seg.type == "standard" then
        local val = self.standardVars[seg.value]
        if val ~= nil then
            return val, nil
        end
        return nil, "standard variable @" .. seg.value .. " not found"

    elseif seg.type == "property" then
        if type(current) == "table" then
            return current[seg.value], nil
        end
        return nil, "cannot get property " .. seg.value .. " from " .. type(current)

    elseif seg.type == "index" then
        if type(current) == "table" then
            -- Lua arrays are already 1-based
            return current[seg.index], nil
        end
        return nil, "cannot index " .. type(current)

    elseif seg.type == "method" then
        if type(current) == "table" then
            local method = current[seg.value]
            if type(method) == "function" then
                return method(current), nil
            end
        end
        return nil, "method " .. seg.value .. " not found"

    elseif seg.type == "parent" then
        return nil, "parent traversal requires context"
    end

    return nil, "unknown segment type: " .. tostring(seg.type)
end

-- Navigate and return parent + key for setting
function PathNavigator:resolveForWrite(root, path)
    local segments = self:parsePath(path)
    if #segments == 0 then
        return nil, nil, nil, "empty path"
    end

    if #segments == 1 then
        local seg = segments[1]
        if seg.type == "property" then
            return root, seg.value, nil, nil
        elseif seg.type == "index" then
            return root, nil, seg.index, nil
        end
        return nil, nil, nil, "cannot write to " .. seg.type
    end

    -- Navigate all but last segment
    local current = root
    for i = 1, #segments - 1 do
        local newVal, err = self:navigateSegment(current, segments[i])
        if err then
            return nil, nil, nil, err
        end
        current = newVal
    end

    local lastSeg = segments[#segments]
    if lastSeg.type == "property" then
        return current, lastSeg.value, nil, nil
    elseif lastSeg.type == "index" then
        return current, nil, lastSeg.index, nil
    end

    return nil, nil, nil, "cannot write to " .. lastSeg.type
end

-- Set a value at a path
function PathNavigator:set(root, path, value)
    local parent, key, index, err = self:resolveForWrite(root, path)
    if err then
        return false, err
    end

    if type(parent) ~= "table" then
        return false, "parent is not a table"
    end

    if key then
        parent[key] = value
    elseif index then
        parent[index] = value
    else
        return false, "no key or index"
    end

    return true, nil
end

return PathNavigator
