-- Contact Manager Demo Backend
-- Spec: demo.md
-- CRC: crc-LuaSession.md
-- Uses automatic change detection - no manual update() calls needed

-- Contact type for individual contacts
Contact = session:prototype("Contact", {
                               firstName = "",
                               lastName = "",
                               email = "",
                               phone = "",
                               notes = "",
                               emergencyContact = nil,  -- reference to another Contact
})

function Contact:fullName()
    local first = self.firstName or ""
    local last = self.lastName or ""
    local result = first .. last
    if first ~= "" and last ~= "" then
       result = first .. " " .. last
    end
    return result
end

function Contact:emergencyContactName()
    if self.emergencyContact then
        return self.emergencyContact:fullName()
    end
    return ""
end

function Contact:hasEmergencyContact()
    return self.emergencyContact ~= nil
end

function Contact:noEmergencyContact()
    return self.emergencyContact == nil
end

-- ContactPresenter - wraps Contact for UI interactions
-- Constructed by ViewList with viewItem: {baseItem, list, index}
ContactPresenter = session:prototype("ContactPresenter", {
                                        viewItem = EMPTY,
                                        contact = EMPTY,
})

function ContactPresenter:new(listItem)
   local tbl = session:create(ContactPresenter, {
      viewItem = listItem,
      contact = listItem.baseItem  -- the domain Contact
   })
   local i = listItem.baseItem
   print("LUA: ContactPresenter on ", i, " type ", i.type, "metatable", getmetatable(i))
   return tbl
end

function ContactPresenter:noPhone()
    return not self.contact.phone or self.contact.phone == ""
end

-- UI actions (app is accessible as upvalue from module scope)
function ContactPresenter:edit()
    contactApp:editContact(self.contact)
end

function ContactPresenter:select()
    contactApp:selectContact(self.contact)
end

function ContactPresenter:delete()
    contactApp:deleteContact(self.contact)
end

-- ContactApp presenter - all view state is inline
ContactApp = session:prototype("ContactApp", {
                                  title = "",
                                  _allContacts = EMPTY,
                                  searchQuery = "",
                                  -- View state
                                  isEditView = false,
                                  isListView = false,
                                  formTitle = "",
                                  error = nil,
                                  -- Edit form fields
                                  editFirstName = "",
                                  editLastName = "",
                                  editEmail = "",
                                  editPhone = "",
                                  editNotes = "",
                                  editEmergencyContactId = "",  -- "" or index string
                                  -- Currently editing contact (nil = creating new)
                                  _editingContact = nil,

})

function ContactApp:new(tbl)
    tbl = session:create(ContactApp, tbl or {})
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
    tbl.editEmergencyContactId = ""
    -- Currently editing contact (nil = creating new)
    tbl._editingContact = nil
    -- Selected contact for detail view (nil = none selected)
    tbl.selectedContact = nil
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
    self.editEmergencyContactId = ""
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
        -- Find the index of the emergency contact in filtered options
        -- Note: Lua index is 1-based, JS expects 0-based
        self.editEmergencyContactId = ""
        if contact.emergencyContact then
            local options = self:emergencyContactOptions()
            for i, c in ipairs(options) do
                if c == contact.emergencyContact then
                    self.editEmergencyContactId = tostring(i - 1)  -- Convert to 0-based
                    break
                end
            end
        end
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

    -- Convert emergency contact id to reference (index into filtered options)
    -- Note: editEmergencyContactId is 0-based (from JS), Lua tables are 1-indexed
    local emergencyContact = nil
    if self.editEmergencyContactId ~= "" then
        local idx = tonumber(self.editEmergencyContactId)
        if idx then
            idx = idx + 1  -- Convert from 0-based (JS) to 1-based (Lua)
        end
        local options = self:emergencyContactOptions()
        if idx and options[idx] then
            emergencyContact = options[idx]
        end
    end

    if self._editingContact then
        -- Update existing contact directly
        self._editingContact.firstName = self.editFirstName
        self._editingContact.lastName = self.editLastName
        self._editingContact.email = self.editEmail
        self._editingContact.phone = self.editPhone
        self._editingContact.notes = self.editNotes
        self._editingContact.emergencyContact = emergencyContact
    else
        -- Create new contact
        local contact = Contact:new({
            firstName = self.editFirstName,
            lastName = self.editLastName,
            email = self.editEmail,
            phone = self.editPhone,
            notes = self.editNotes,
            emergencyContact = emergencyContact
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

-- Select a contact for detail view
function ContactApp:selectContact(contact)
    self.selectedContact = contact
end

-- Deselect the current contact
function ContactApp:deselectContact()
    self.selectedContact = nil
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

-- Get emergency contact options for dropdown (excludes contact being edited)
-- Returns filtered contacts list; ViewList provides index via ViewListItem
function ContactApp:emergencyContactOptions()
    local options = {}
    for _, contact in ipairs(self._allContacts) do
        if contact ~= self._editingContact then
            table.insert(options, contact)
        end
    end
    return options
end


if not session.reloading then
   print("LUA: initialized. Contact", Contact, "ContactPresenter", ContactPresenter, "ContactApp", ContactApp)
   -- Create the app instance
   contactApp = ContactApp:new({
         title = "Contact Manager"
   })
   
   -- Register as app variable (the ONLY variable backend creates)
   session:createAppVariable(contactApp)
   
   -- Add sample contacts to master list
   table.insert(contactApp._allContacts, Contact:new({
                      firstName = "Alice", lastName = "Smith",
                      email = "alice@example.com", phone = "555-0101",
                      notes = "Met at conference"
   }))
   table.insert(contactApp._allContacts, Contact:new({
                      firstName = "Bob", lastName = "Johnson",
                      email = "bob@example.com", phone = "555-0102"
   }))
   table.insert(contactApp._allContacts, Contact:new({
                      firstName = "Carol", lastName = "Williams",
                      email = "carol@example.com", phone = "555-0103",
                      notes = "Project lead"
   }))
   
   ui.log("Contact Manager initialized for session")
end

x = 123
print("X = "..x)
