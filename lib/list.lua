-- CRC: crc-ListPresenter.md
-- Spec: main.md
-- List presenter for managing arrays of items

local Presenter = require("presenter")

local ListPresenter = setmetatable({}, {__index = Presenter})
ListPresenter.__index = ListPresenter

-- Create list presenter type
function ListPresenter.new()
    local self = setmetatable({}, ListPresenter)
    self._type = "List"
    self.items = {}
    self.selectedId = nil

    return self
end

-- Initialize list presenter
function ListPresenter:init()
    self.items = {}
    self.selectedId = nil
end

-- Add an item to the list
function ListPresenter:addItem(variableId)
    table.insert(self.items, variableId)
    self:notifyChange()
end

-- Remove an item from the list
function ListPresenter:removeItem(variableId)
    for i, id in ipairs(self.items) do
        if id == variableId then
            table.remove(self.items, i)
            if self.selectedId == variableId then
                self.selectedId = nil
            end
            self:notifyChange()
            return true
        end
    end
    return false
end

-- Select an item
function ListPresenter:selectItem(variableId)
    for _, id in ipairs(self.items) do
        if id == variableId then
            self.selectedId = variableId
            self:notifyChange()
            return true
        end
    end
    return false
end

-- Get selected item
function ListPresenter:getSelected()
    return self.selectedId
end

-- Get all items
function ListPresenter:getItems()
    local copy = {}
    for i, id in ipairs(self.items) do
        copy[i] = id
    end
    return copy
end

-- Get item count
function ListPresenter:count()
    return #self.items
end

-- Clear all items
function ListPresenter:clear()
    self.items = {}
    self.selectedId = nil
    self:notifyChange()
end

-- Move item to new position
function ListPresenter:moveItem(fromIndex, toIndex)
    if fromIndex < 1 or fromIndex > #self.items then
        return false
    end
    if toIndex < 1 or toIndex > #self.items then
        return false
    end

    local item = table.remove(self.items, fromIndex)
    table.insert(self.items, toIndex, item)
    self:notifyChange()
    return true
end

-- Convert to data for variable
function ListPresenter:toData()
    return {
        items = self.items,
        selectedId = self.selectedId,
        count = #self.items
    }
end

-- Register with runtime
ui.registerPresenter("List", ListPresenter)

return ListPresenter
