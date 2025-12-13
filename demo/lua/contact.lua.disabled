-- Contact Manager Demo Backend
-- Spec: demo.md
-- CRC: crc-LuaSession.md
-- Uses automatic change detection - no manual update() calls needed

-- Contact type for individual contacts
local Contact = {type = "Contact"}
Contact.__index = Contact

function Contact:new(tbl)
    tbl = tbl or {}
    setmetatable(tbl, self)
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

function Contact:noPhone()
    return not self.phone or self.phone == ""
end

-- ContactPresenter - wraps Contact for UI interactions
-- Constructed by ViewList with viewItem: {baseItem, list, index}
local ContactPresenter = {type = "ContactPresenter"}
ContactPresenter.__index = ContactPresenter

function ContactPresenter:new(viewItem)
    local tbl = {
        viewItem = viewItem,
        contact = viewItem.baseItem  -- the domain Contact
    }
    setmetatable(tbl, self)
    return tbl
end

-- Delegate to domain object for display
function ContactPresenter:fullName()
    return self.contact:fullName()
end

function ContactPresenter:email()
    return self.contact.email
end

function ContactPresenter:phone()
    return self.contact.phone
end

function ContactPresenter:noPhone()
    return self.contact:noPhone()
end

-- UI actions (app is accessible as upvalue from module scope)
function ContactPresenter:edit()
    app:editContact(self.contact)
end

function ContactPresenter:delete()
    app:deleteContact(self.contact)
end

-- ContactApp presenter - all view state is inline
local ContactApp = {type = "ContactApp"}
ContactApp.__index = ContactApp

function ContactApp:new(tbl)
    tbl = tbl or {}
    setmetatable(tbl, self)
    tbl.title = tbl.title or "Contact Manager"
    tbl.contacts = tbl.contacts or {}
    tbl.hasContacts = #tbl.contacts > 0
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
        table.insert(self.contacts, contact)
        self.hasContacts = true
    end

    self.isEditView = false
    self.isListView = true
    self.error = nil
end

-- Delete a contact
function ContactApp:deleteContact(contact)
    if contact then
        -- Find and remove by reference
        for i, c in ipairs(self.contacts) do
            if c == contact then
                table.remove(self.contacts, i)
                break
            end
        end
        self.hasContacts = #self.contacts > 0
    end
end

-- Cancel editing
function ContactApp:cancelEdit()
    self.isEditView = false
    self.isListView = true
    self.error = nil
end

-- Search contacts (placeholder - would filter in real app)
function ContactApp:search(query)
    -- For now just store the query
    self.searchQuery = query or ""
end

-- Get contact count
function ContactApp:contactCount()
    return #self.contacts
end

-- Create the app instance
local app = ContactApp:new({
    title = "Contact Manager"
})

-- Register as app variable (the ONLY variable backend creates)
session:createAppVariable(app)

-- Add sample contacts
table.insert(app.contacts, Contact:new({
    firstName = "Alice", lastName = "Smith",
    email = "alice@example.com", phone = "555-0101",
    notes = "Met at conference"
}))
table.insert(app.contacts, Contact:new({
    firstName = "Bob", lastName = "Johnson",
    email = "bob@example.com", phone = "555-0102"
}))
table.insert(app.contacts, Contact:new({
    firstName = "Carol", lastName = "Williams",
    email = "carol@example.com", phone = "555-0103",
    notes = "Project lead"
}))
app.hasContacts = true

ui.log("Contact Manager initialized for session")
