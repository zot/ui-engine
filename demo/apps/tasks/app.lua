-- Tasks App

-- Task prototype
Task = session:prototype("Task", {
    name = "",
    description = "",
    completed = false,
    editing = false,
    expanded = false,
    editName = "",
    editDesc = ""
})

function Task:new(instance)
    instance = session:create(Task, instance)
    instance.name = instance.name or ""
    instance.description = instance.description or ""
    instance.completed = instance.completed or false
    instance.editing = false
    instance.expanded = false
    instance.editName = ""
    instance.editDesc = ""
    return instance
end

function Task:toggle()
    self.completed = not self.completed
end

function Task:startEdit()
    self.editing = true
    self.editName = self.name
end

function Task:saveEdit()
    self.name = self.editName
    self.editing = false
end

function Task:cancelEdit()
    self.editing = false
    self.editName = ""
end

function Task:toggleExpand()
    if self.expanded then
        self.expanded = false
    else
        self.expanded = true
        self.editName = self.name
        self.editDesc = self.description
    end
end

function Task:saveDetails()
    self.name = self.editName
    self.description = self.editDesc
    self.expanded = false
end

function Task:cancelDetails()
    self.expanded = false
    self.editName = ""
    self.editDesc = ""
end

function Task:isSelected()
    return tasks and tasks.selected == self
end

function Task:select()
    if tasks then
        tasks:selectTask(self)
    end
end

function Task:isNotEditing()
    return not self.editing
end

function Task:isExpanded()
    return self.expanded
end

function Task:isCollapsed()
    return not self.expanded
end

-- TasksApp prototype
TasksApp = session:prototype("TasksApp", {
    tasks = EMPTY,
    search = "",
    selected = EMPTY
})

function TasksApp:new(instance)
    instance = session:create(TasksApp, instance)
    instance.tasks = instance.tasks or {}
    instance.search = ""
    instance.selected = nil
    return instance
end

function TasksApp:filteredTasks()
    if self.search == "" then
        return self.tasks
    end
    local result = {}
    local searchLower = string.lower(self.search)
    for _, task in ipairs(self.tasks) do
        if string.find(string.lower(task.name), searchLower, 1, true) then
            table.insert(result, task)
        end
    end
    return result
end

function TasksApp:addTask()
    local newTask = Task:new({ name = "" })

    if self.selected then
        -- Insert after selected task
        for i, task in ipairs(self.tasks) do
            if task == self.selected then
                table.insert(self.tasks, i + 1, newTask)
                break
            end
        end
    else
        -- Insert at end
        table.insert(self.tasks, newTask)
    end

    self.selected = newTask
    newTask:startEdit()  -- Start editing immediately
end

function TasksApp:selectTask(task)
    self.selected = task
end

-- Instance creation (idempotent)
if not session.reloading then
    tasks = TasksApp:new()
    -- Add sample tasks
    table.insert(tasks.tasks, Task:new({ name = "First task", description = "This is the first task" }))
    table.insert(tasks.tasks, Task:new({ name = "Second task", completed = true }))
    table.insert(tasks.tasks, Task:new({ name = "Third task", description = "Another task with details" }))
    session:createAppVariable(tasks)
end
