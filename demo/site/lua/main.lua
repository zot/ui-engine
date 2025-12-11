-- Contact Manager Demo Backend
-- Spec: demo.md
-- CRC: crc-LuaSession.md

-- ContactApp presenter with methods callable via ui-action paths
local ContactApp = {}
ContactApp.__index = ContactApp

function ContactApp:new()
    local self = setmetatable({}, ContactApp)
    self.contacts = {}
    self.nextId = 1
    return self
end

-- Add a new contact
function ContactApp:addContact()
    -- Show empty form for new contact
    local app = session:getAppVariable()
    app:update({
        currentView = "edit",
        selectedContact = nil,
        error = nil
    })
end

-- Edit an existing contact
function ContactApp:editContact(contactId)
    local app = session:getAppVariable()
    if contactId then
        app:update({
            currentView = "edit",
            selectedContact = {obj = tonumber(contactId)},
            error = nil
        })
    end
end

-- Save contact (create or update)
function ContactApp:saveContact(formData)
    local app = session:getAppVariable()
    local appValue = app:getValue()

    -- Parse form data if it's a string
    local data = formData
    if type(formData) == "string" then
        -- formData might be JSON-like
        data = {
            firstName = formData.firstName,
            lastName = formData.lastName,
            email = formData.email,
            phone = formData.phone,
            notes = formData.notes
        }
    end

    -- Check if we're editing or creating
    local contactId = data and data.contactId

    local result, err
    if contactId then
        -- Update existing
        result, err = self:updateContactData(tonumber(contactId), data)
    else
        -- Create new
        result, err = self:createContactData(data)
    end

    if err then
        app:update({error = err})
    else
        app:update({
            currentView = "list",
            selectedContact = nil,
            error = nil
        })
    end
end

-- Delete a contact
function ContactApp:deleteContact(contactId)
    local app = session:getAppVariable()
    if contactId then
        local success, err = self:deleteContactData(tonumber(contactId))
        if err then
            app:update({error = err})
        end
    end
end

-- Cancel editing
function ContactApp:cancelEdit()
    local app = session:getAppVariable()
    app:update({
        currentView = "list",
        selectedContact = nil,
        error = nil
    })
end

-- Search contacts
function ContactApp:search(query)
    local app = session:getAppVariable()
    app:update({searchQuery = query or ""})
    -- Note: Filtering is done client-side based on searchQuery
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

    local id = self.nextId
    self.nextId = self.nextId + 1

    local contact = {
        id = id,
        type = "Contact",
        view = "contact-item",
        firstName = data.firstName or "",
        lastName = data.lastName or "",
        email = data.email or "",
        phone = data.phone or "",
        notes = data.notes or ""
    }

    self.contacts[id] = contact

    -- Create variable for contact
    local app = session:getAppVariable()
    local contactVar = session:createVariable(app:getId(), contact)

    -- Add to app's contact list
    local appValue = app:getValue()
    local contactList = appValue.contacts or {}
    table.insert(contactList, {obj = contactVar:getId()})
    app:update({contacts = contactList})

    return contact
end

-- Internal: Update contact data
function ContactApp:updateContactData(id, data)
    local contact = self.contacts[id]
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

    -- Update fields
    for key, value in pairs(data) do
        if key ~= "id" and key ~= "type" and key ~= "view" and key ~= "contactId" then
            contact[key] = value
        end
    end

    -- Update variable
    local contactVar = session:getVariable(id)
    if contactVar then
        contactVar:update(contact)
    end

    return contact
end

-- Internal: Delete contact data
function ContactApp:deleteContactData(id)
    local contact = self.contacts[id]
    if not contact then
        return false, "Contact not found"
    end

    self.contacts[id] = nil

    -- Remove from app's contact list
    local app = session:getAppVariable()
    local appValue = app:getValue()
    local contactList = appValue.contacts or {}
    local newList = {}

    for _, ref in ipairs(contactList) do
        if ref.obj ~= id then
            table.insert(newList, ref)
        end
    end

    app:update({contacts = newList})

    -- Destroy the variable
    session:destroyVariable(id)

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

-- Create the presenter
local presenter = ContactApp:new()

-- Create the app variable (variable 1) with presenter
local app = session:createAppVariable({
    type = "ContactApp",
    view = "contact-app",
    title = "Contact Manager",
    currentView = "list",
    contacts = {},
    selectedContact = nil,
    searchQuery = "",
    error = nil,
    presenter = presenter  -- ui-action="presenter.addContact()" calls this
}, {
    type = "ContactApp"
})

-- Add sample contacts
presenter:addSampleContacts()

ui.log("Contact Manager initialized for session")
