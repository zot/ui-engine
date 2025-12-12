-- ViewList wrapper library for Lua
-- CRC: crc-ViewList.md, crc-ViewItem.md, crc-Wrapper.md
-- Spec: viewdefs.md, protocol.md
-- Sequence: seq-viewlist-presenter-sync.md, seq-wrapper-transform.md
--
-- This library provides helper functions for working with ViewList wrappers.
-- The ViewList wrapper is built-in (implemented in Go) and automatically
-- registered. This library provides utilities for:
-- - Creating ViewItem-compatible presenters
-- - Helper functions for list manipulation
--
-- Usage:
--   local viewlist = require("viewlist")
--
--   -- Create an ItemWrapper presenter type
--   local ContactPresenter = viewlist.createItemWrapper("ContactPresenter", {
--       init = function(self)
--           -- self.viewItem is set by ViewList
--           -- self.viewItem.baseItem is the domain object ref
--       end,
--
--       delete = function(self)
--           self.viewItem:remove()
--       end
--   })

local ViewList = {}

-- Create an ItemWrapper presenter type.
-- ItemWrappers receive a viewItem in their init method with:
-- - viewItem.baseItem: reference to the domain object ({obj: ID})
-- - viewItem.index: position in the list (0-based)
-- - viewItem:remove(): removes this item from the list
--
-- @param name The presenter type name
-- @param methods Table of methods for the presenter
-- @return The presenter type table
function ViewList.createItemWrapper(name, methods)
    local ItemWrapper = {}
    ItemWrapper.__index = ItemWrapper
    ItemWrapper._type = name

    -- Copy provided methods
    for k, v in pairs(methods or {}) do
        ItemWrapper[k] = v
    end

    -- Ensure init wraps any provided init
    local userInit = methods and methods.init
    function ItemWrapper:init()
        -- viewItem is set by Go's CreateItemWrapper before init is called
        if userInit then
            userInit(self)
        end
    end

    -- Register with the runtime
    ui.registerPresenter(name, ItemWrapper)

    return ItemWrapper
end

-- Helper to get the base domain object from a ViewItem
-- @param viewItem The ViewItem table
-- @return The domain object reference ({obj: ID})
function ViewList.getBaseItem(viewItem)
    if viewItem then
        return viewItem.baseItem
    end
    return nil
end

-- Helper to get the index of a ViewItem in its list
-- @param viewItem The ViewItem table
-- @return The 0-based index
function ViewList.getIndex(viewItem)
    if viewItem then
        return viewItem.index
    end
    return -1
end

-- Helper to remove a ViewItem from its list
-- @param viewItem The ViewItem table
function ViewList.remove(viewItem)
    if viewItem and viewItem.remove then
        viewItem:remove()
    end
end

return ViewList
