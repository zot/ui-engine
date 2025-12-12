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

-- ContactApp presenter - all view state is inline (no separate view types)
local ContactApp = {type = "ContactApp"}
ContactApp.__index = ContactApp

function ContactApp:new(tbl)
    tbl = tbl or {}
    setmetatable(tbl, self)
    -- App-level state
    tbl.title = tbl.title or "Contact Manager"
    tbl.currentView = tbl.currentView or "list"
    tbl.isEditView = tbl.isEditView or false
    tbl.isListView = tbl.isListView ~= false  -- default true
    tbl.error = tbl.error or nil
    -- List view state (inline)
    tbl.searchQuery = tbl.searchQuery or ""
    tbl.contacts = tbl.contacts or {}
    tbl.hasContacts = tbl.hasContacts or false
    -- Edit view state (inline)
    tbl.formTitle = tbl.formTitle or "Add Contact"
    tbl.editFirstName = tbl.editFirstName or ""
    tbl.editLastName = tbl.editLastName or ""
    tbl.editEmail = tbl.editEmail or ""
    tbl.editPhone = tbl.editPhone or ""
    tbl.editNotes = tbl.editNotes or ""
    tbl.editContactId = tbl.editContactId or nil
    -- Internal state (not serialized - prefixed with _)
    tbl._contactData = {}
    tbl._nextId = 1
    return tbl
end

-- Add a new contact - switches to edit view
function ContactApp:addContact()
    -- Just modify self directly - changes auto-detected after message processing
    self.currentView = "edit"
    self.isEditView = true
    self.isListView = false
    self.formTitle = "Add Contact"
    self.editFirstName = ""
    self.editLastName = ""
    self.editEmail = ""
    self.editPhone = ""
    self.editNotes = ""
    self.editContactId = nil
    self.error = nil
end

-- Edit an existing contact
function ContactApp:editContact(contactId)
    if contactId then
        local contact = self._contactData[tonumber(contactId)]
        if contact then
            self.currentView = "edit"
            self.isEditView = true
            self.isListView = false
            self.formTitle = "Edit Contact"
            self.editFirstName = contact.firstName or ""
            self.editLastName = contact.lastName or ""
            self.editEmail = contact.email or ""
            self.editPhone = contact.phone or ""
            self.editNotes = contact.notes or ""
            self.editContactId = tonumber(contactId)
            self.error = nil
        end
    end
end

-- Save contact (create or update)
function ContactApp:saveContact()
    -- Get data from edit fields (self is the app object)
    local data = {
        firstName = self.editFirstName,
        lastName = self.editLastName,
        email = self.editEmail,
        phone = self.editPhone,
        notes = self.editNotes
    }

    -- Check if we're editing or creating
    local contactId = self.editContactId

    local result, err
    if contactId then
        result, err = self:updateContactData(contactId, data)
    else
        result, err = self:createContactData(data)
    end

    if err then
        self.error = err
    else
        self.currentView = "list"
        self.isEditView = false
        self.isListView = true
        self.error = nil
    end
end

-- Delete a contact
function ContactApp:deleteContact(contactId)
    if contactId then
        local success, err = self:deleteContactData(tonumber(contactId))
        if err then
            self.error = err
        end
    end
end

-- Cancel editing
function ContactApp:cancelEdit()
    self.currentView = "list"
    self.isEditView = false
    self.isListView = true
    self.error = nil
end

-- Search contacts
function ContactApp:search(query)
    self.searchQuery = query or ""
end

-- Internal: Create contact data
function ContactApp:createContactData(data)
    -- Validate required fields
    if not data or not data.firstName or data.firstName == "" then
        return nil, "First name is required"
    end
    if not data.email or data.email == "" then
        return nil, "Email is required"
    end

    local id = self._nextId
    self._nextId = self._nextId + 1

    local contact = Contact:new({
        id = id,
        firstName = data.firstName or "",
        lastName = data.lastName or "",
        email = data.email or "",
        phone = data.phone or "",
        notes = data.notes or ""
    })

    self._contactData[id] = contact

    -- Create variable for contact - pass self (the app object) as parent
    -- The variable references the contact object for change detection
    local contactVarId = session:createVariable(self, contact)

    -- Add to app's contact list (direct modification)
    table.insert(self.contacts, {obj = contactVarId})
    self.hasContacts = true

    return contact
end

-- Internal: Update contact data
function ContactApp:updateContactData(id, data)
    local contact = self._contactData[id]
    if not contact then
        return nil, "Contact not found"
    end

    -- Validate required fields
    if data.firstName ~= nil and data.firstName == "" then
        return nil, "First name is required"
    end
    if data.email ~= nil and data.email == "" then
        return nil, "Email is required"
    end

    -- Update fields directly on the contact object
    -- Changes are auto-detected because the contact is a watched variable
    for key, value in pairs(data) do
        if key ~= "id" and key ~= "type" and key ~= "contactId" then
            contact[key] = value
        end
    end

    return contact
end

-- Internal: Delete contact data
function ContactApp:deleteContactData(id)
    local contact = self._contactData[id]
    if not contact then
        return false, "Contact not found"
    end

    self._contactData[id] = nil

    -- Remove from app's contact list (direct modification)
    local newList = {}
    for _, ref in ipairs(self.contacts) do
        if ref.obj ~= id then
            table.insert(newList, ref)
        end
    end
    self.contacts = newList
    self.hasContacts = #newList > 0

    -- Destroy the variable (can pass the contact object directly)
    session:destroyVariable(contact)

    return true
end

-- Add sample contacts for demo
function ContactApp:addSampleContacts()
    local samples = {
        {firstName = "Alice", lastName = "Smith", email = "alice@example.com", phone = "555-0101", notes = "Met at conference"},
        {firstName = "Bob", lastName = "Johnson", email = "bob@example.com", phone = "555-0102", notes = ""},
        {firstName = "Carol", lastName = "Williams", email = "carol@example.com", phone = "555-0103", notes = "Project lead"}
    }

    for _, contact in ipairs(samples) do
        self:createContactData(contact)
    end
end

-- Create the app instance and register as app variable
-- The type property is automatically extracted from the metatable's type field
local app = ContactApp:new({
    title = "Contact Manager"
})

session:createAppVariable(app)

-- Add sample contacts
app:addSampleContacts()

ui.log("Contact Manager initialized for session")
