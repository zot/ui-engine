# Tasks App Design

## Intent

A task list with inline editing, selection tracking, search filtering, and detail editing.

## Layout

```
+------------------------------------------+
| Tasks: [search_________]  [+]            |
+------------------------------------------+
| [ ] Task name                      [Edit]|  <- selected (highlighted)
| [x] Another task (strikethrough)   [Edit]|
| [ ] Third task                     [Edit]|
|   +------------------------------------+ |  <- detail panel (expanded)
|   | Name: [___________]                | |
|   | Description:                       | |
|   | [____________________________]     | |
|   | Status: [ ] Completed              | |
|   | [Save] [Cancel]                    | |
|   +------------------------------------+ |
+------------------------------------------+
```

## Data Model

### TasksApp

| Field       | Type     | Description                          |
|-------------|----------|--------------------------------------|
| tasks       | Task[]   | All tasks                            |
| search      | string   | Current search filter text           |
| selected    | Task     | Currently selected task (or nil)     |

### Task

| Field       | Type     | Description                          |
|-------------|----------|--------------------------------------|
| name        | string   | Task title                           |
| description | string   | Longer task description              |
| completed   | boolean  | Whether task is done                 |
| editing     | boolean  | Whether inline name edit is active   |
| expanded    | boolean  | Whether detail panel is open         |
| editName    | string   | Temp name while editing inline       |
| editDesc    | string   | Temp description in detail panel     |

## Methods

### TasksApp

| Method           | Description                                      |
|------------------|--------------------------------------------------|
| filteredTasks()  | Returns tasks matching search filter             |
| addTask()        | Creates new task below selected or at end        |
| selectTask(task) | Sets selected task                               |

### Task

| Method           | Description                                      |
|------------------|--------------------------------------------------|
| toggle()         | Toggle completed status                          |
| startEdit()      | Begin inline name editing                        |
| saveEdit()       | Save inline name edit                            |
| cancelEdit()     | Cancel inline name edit                          |
| toggleExpand()   | Open/close detail panel                          |
| saveDetails()    | Save detail panel changes                        |
| cancelDetails()  | Cancel detail panel, close it                    |
| isSelected()     | Check if this task is selected                   |

## ViewDefs

| File                        | Type     | Description              |
|-----------------------------|----------|--------------------------|
| TasksApp.DEFAULT.html       | TasksApp | Main app layout          |
| Task.list-item.html         | Task     | Task row with detail     |

## Events

None needed - all interactions handled in Lua.
