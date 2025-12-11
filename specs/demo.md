# Demo: Contact Manager

A simple contact manager demonstrating the UI platform capabilities with a Lua backend.

## Overview

The contact manager allows users to:
- View a list of contacts
- Add new contacts
- Edit existing contacts
- Delete contacts
- Search/filter contacts

All data is stored in-memory within the Lua backend.

## Data Model

### Contact

```
{
  id: number,
  firstName: string,
  lastName: string,
  email: string,
  phone: string,
  notes: string
}
```

## UI Structure

### App Shell (Variable 1)

```
{
  type: "App",
  view: "contact-app",
  title: "Contact Manager",
  currentView: "list" | "edit",
  contacts: [{obj: contactId}, ...],
  selectedContact: {obj: contactId} | null,
  searchQuery: string,
  error: string | null
}
```

### Contact Variable

```
{
  type: "Contact",
  view: "contact-item",
  ...contact fields
}
```

## Views

### contact-app (Main App View)

- Header with app title and search box
- "Add Contact" button
- Contact list (filterable by search)
- Edit form (shown when editing/adding)

### contact-item (List Item)

- Display name (firstName lastName)
- Email
- Edit and Delete buttons

### contact-form (Edit/Add Form)

- Input fields for all contact properties
- Save and Cancel buttons
- Validation feedback

## Actions

| Action | Trigger | Behavior |
|--------|---------|----------|
| `addContact()` | Click "Add" button | Create new empty contact, show form |
| `editContact()` | Click contact's Edit | Set selectedContact, show form |
| `saveContact()` | Click Save in form | Validate and save, return to list |
| `deleteContact()` | Click Delete | Remove contact from list |
| `cancelEdit()` | Click Cancel | Clear selection, return to list |
| `search()` | Type in search box | Filter displayed contacts |

## Lua Backend

The Lua backend (`lua/contacts.lua`) handles:

1. **Initialization**: Creates the app presenter with empty contact list
2. **Contact CRUD**: Responds to create/update/delete actions
3. **Search**: Filters contacts based on query (matches name, email, phone)
4. **Validation**: Ensures required fields (firstName, email) are present

## File Structure

```
demo/
  lua/
    contacts.lua     # Backend logic
  site/
    index.html       # Demo page with viewdefs
    embed.go         # Go embed directive
```

## Usage

Start the server with the demo directory:

```bash
ui serve --dir demo/site --lua-path demo/lua
```

Then open browser to `http://localhost:8080`

The app will:
1. Create a new session
2. Initialize the Lua contact manager backend
3. Display the contact list with sample data
4. Allow CRUD operations on contacts

## Build Integration

For release builds, the demo files are located in the `demo/` directory, separate from the main `site/` directory used by the core platform.

To build a demo-specific binary:
```bash
make demo
```

This creates a standalone binary with the demo embedded.
