-- CRC: crc-AppPresenter.md
-- Spec: main.md
-- App presenter for session-level state

local Presenter = require("presenter")

local AppPresenter = setmetatable({}, {__index = Presenter})
AppPresenter.__index = AppPresenter

-- Create app presenter type
function AppPresenter.new()
    local self = setmetatable({}, AppPresenter)
    self._type = "App"
    self.history = {}
    self.historyIndex = 0
    self.currentUrl = nil
    self.currentVariableId = nil

    return self
end

-- Initialize app presenter
function AppPresenter:init()
    self.history = {}
    self.historyIndex = 0
end

-- Navigate to a URL with associated variable
function AppPresenter:navigate(url, variableId)
    -- Truncate forward history
    while #self.history > self.historyIndex do
        table.remove(self.history)
    end

    -- Add new entry
    table.insert(self.history, {
        url = url,
        variableId = variableId
    })
    self.historyIndex = #self.history

    self.currentUrl = url
    self.currentVariableId = variableId

    self:notifyChange()
end

-- Go back in history
function AppPresenter:back()
    if self.historyIndex > 1 then
        self.historyIndex = self.historyIndex - 1
        local entry = self.history[self.historyIndex]
        self.currentUrl = entry.url
        self.currentVariableId = entry.variableId
        self:notifyChange()
        return true
    end
    return false
end

-- Go forward in history
function AppPresenter:forward()
    if self.historyIndex < #self.history then
        self.historyIndex = self.historyIndex + 1
        local entry = self.history[self.historyIndex]
        self.currentUrl = entry.url
        self.currentVariableId = entry.variableId
        self:notifyChange()
        return true
    end
    return false
end

-- Go to specific history index
function AppPresenter:go(index)
    if index >= 1 and index <= #self.history then
        self.historyIndex = index
        local entry = self.history[self.historyIndex]
        self.currentUrl = entry.url
        self.currentVariableId = entry.variableId
        self:notifyChange()
        return true
    end
    return false
end

-- Replace current page without adding history
function AppPresenter:replaceCurrentPage(url, variableId)
    if self.historyIndex > 0 then
        self.history[self.historyIndex] = {
            url = url,
            variableId = variableId
        }
    else
        table.insert(self.history, {
            url = url,
            variableId = variableId
        })
        self.historyIndex = 1
    end

    self.currentUrl = url
    self.currentVariableId = variableId

    self:notifyChange()
end

-- Get current page
function AppPresenter:currentPage()
    if self.historyIndex > 0 then
        return self.history[self.historyIndex]
    end
    return nil
end

-- Convert to data for variable
function AppPresenter:toData()
    return {
        url = self.currentUrl,
        variableId = self.currentVariableId,
        canGoBack = self.historyIndex > 1,
        canGoForward = self.historyIndex < #self.history,
        historyLength = #self.history
    }
end

-- Register with runtime
ui.registerPresenter("App", AppPresenter)

return AppPresenter
