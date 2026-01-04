-- Contact Manager Demo Backend
-- Spec: demo.md
-- CRC: crc-LuaSession.md
-- Uses automatic change detection - no manual update() calls needed

local app

-- Contact type for individual contacts
Contact = {type = "Contact"}
Contact.__index = Contact

function Contact:new(tbl)
   tbl = tbl or {}
   setmetatable(tbl, self)
   print("LUA: made Contact", tbl, "type", tbl.type, "metatable", getmetatable(tbl))
   return tbl
end

function Contact:fullName()
    local first = self.firstName or ""
    local last = self.lastName or ""
    if first ~= "" and last ~= "" then
        return first .. " " .. last
    end
    return first .. last
end

-- ContactPresenter - wraps Contact for UI interactions
-- Constructed by ViewList with viewItem: {baseItem, list, index}
ContactPresenter = {type = "ContactPresenter"}
ContactPresenter.__index = ContactPresenter

function ContactPresenter:new(listItem)
   local tbl = {
      viewItem = listItem,
      contact = listItem.baseItem  -- the domain Contact
   }
   setmetatable(tbl, self)
   local i = listItem.baseItem
   print("LUA: ContactPresenter on ", i, " type ", i.type, "metatable", getmetatable(i))
   return tbl
end

function ContactPresenter:noPhone()
    return not self.contact.phone or self.contact.phone == ""
end

-- UI actions (app is accessible as upvalue from module scope)
function ContactPresenter:edit()
    app:editContact(self.contact)
end

function ContactPresenter:delete()
    app:deleteContact(self.contact)
end

-- ContactApp presenter - all view state is inline
ContactApp = {type = "ContactApp"}
ContactApp.__index = ContactApp

function ContactApp:new(tbl)
    tbl = tbl or {}
    setmetatable(tbl, self)
    tbl.title = tbl.title or "Contact Manager"
    tbl._allContacts = tbl._allContacts or {}  -- Master list
    tbl.searchQuery = ""
    -- View state
    tbl.isEditView = false
    tbl.isListView = true
    tbl.formTitle = "Add Contact"
    tbl.error = nil
    -- Edit form fields
    tbl.editFirstName = ""
    tbl.editLastName = ""
    tbl.editEmail = ""
    tbl.editPhone = ""
    tbl.editNotes = ""
    -- Currently editing contact (nil = creating new)
    tbl._editingContact = nil
    return tbl
end

-- Add a new contact - switch to edit view
function ContactApp:addContact()
    self.isEditView = true
    self.isListView = false
    self.formTitle = "Add Contact"
    self.editFirstName = ""
    self.editLastName = ""
    self.editEmail = ""
    self.editPhone = ""
    self.editNotes = ""
    self._editingContact = nil
    self.error = nil
end

-- Edit an existing contact
function ContactApp:editContact(contact)
    if contact then
        self.isEditView = true
        self.isListView = false
        self.formTitle = "Edit Contact"
        self.editFirstName = contact.firstName or ""
        self.editLastName = contact.lastName or ""
        self.editEmail = contact.email or ""
        self.editPhone = contact.phone or ""
        self.editNotes = contact.notes or ""
        self._editingContact = contact
        self.error = nil
    end
end

-- Save contact (create or update)
function ContactApp:saveContact()
    -- Validate
    if self.editFirstName == "" then
        self.error = "First name is required"
        return
    end
    if self.editEmail == "" then
        self.error = "Email is required"
        return
    end

    if self._editingContact then
        -- Update existing contact directly
        self._editingContact.firstName = self.editFirstName
        self._editingContact.lastName = self.editLastName
        self._editingContact.email = self.editEmail
        self._editingContact.phone = self.editPhone
        self._editingContact.notes = self.editNotes
    else
        -- Create new contact
        local contact = Contact:new({
            firstName = self.editFirstName,
            lastName = self.editLastName,
            email = self.editEmail,
            phone = self.editPhone,
            notes = self.editNotes
        })
        table.insert(self._allContacts, contact)
    end

    self.isEditView = false
    self.isListView = true
    self.error = nil
end

-- Delete a contact
function ContactApp:deleteContact(contact)
    if contact then
        -- Find and remove from master list
        for i, c in ipairs(self._allContacts) do
            if c == contact then
                table.remove(self._allContacts, i)
                break
            end
        end
    end
end

-- Cancel editing
function ContactApp:cancelEdit()
    self.isEditView = false
    self.isListView = true
    self.error = nil
end

-- Filter contacts based on search query (returns filtered array)
function ContactApp:contacts()
    local query = (self.searchQuery or ""):lower()
    local result = {}

    for _, contact in ipairs(self._allContacts) do
        if query == "" then
            table.insert(result, contact)
        else
            -- Search in name and email
            local fullName = contact:fullName():lower()
            local email = (contact.email or ""):lower()
            if fullName:find(query, 1, true) or email:find(query, 1, true) then
                table.insert(result, contact)
            end
        end
    end

    return result
end

-- Get contact count
function ContactApp:contactCount()
    return #self:contacts()
end

-- Check if has contacts
function ContactApp:hasContacts()
    return #self:contacts() > 0
end

-- Select first contact in filtered list (for Enter key in search)
function ContactApp:selectFirstContact()
    local filtered = self:contacts()
    if #filtered > 0 then
        self:editContact(filtered[1])
    end
end

print("LUA: initialized. Contact", Contact, "ContactPresenter", ContactPresenter, "ContactApp", ContactApp)

-- Create the app instance
app = ContactApp:new({
    title = "Contact Manager"
})

-- Register as app variable (the ONLY variable backend creates)
session:createAppVariable(app)

-- Add sample contacts to master list
table.insert(app._allContacts, Contact:new({
    firstName = "Alice", lastName = "Smith",
    email = "alice@example.com", phone = "555-0101",
    notes = "Met at conference"
}))
table.insert(app._allContacts, Contact:new({
    firstName = "Bob", lastName = "Johnson",
    email = "bob@example.com", phone = "555-0102"
}))
table.insert(app._allContacts, Contact:new({
    firstName = "Carol", lastName = "Williams",
    email = "carol@example.com", phone = "555-0103",
    notes = "Project lead"
}))

ui.log("Contact Manager initialized for session")
