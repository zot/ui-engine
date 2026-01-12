# Tasks App Requirements

A simple task management app for viewing and editing a list of tasks.

## Overview

The app displays a list of tasks with inline editing capabilities. Users can quickly edit task names directly in the list, or open a detail view to edit additional task properties.

## Header

The header contains:
- Title "Tasks:"
- A dynamic search field that filters tasks as you type
- A "+" button that creates a new blank task

## Task Properties

Each task has:
- Name (required) - the task title, editable inline
- Description (optional) - longer text describing the task
- Status - whether the task is pending or completed

## List View

The main view shows all tasks in a vertical list. The list tracks which task is currently selected (highlighted). Each task row displays:
- A checkbox to toggle completion status
- The task name, which can be edited inline by clicking on it
- An edit button that opens the detail editor

Clicking a task row selects it.

## Detail Editor

When the edit button is clicked, a detail panel appears (either inline expanded or as a side panel) showing:
- Task name (editable)
- Description (editable textarea)
- Status toggle
- Save and Cancel buttons

## Interactions

- Clicking a task row selects it (visual highlight)
- Clicking a task name makes it editable inline (press Enter to save, Escape to cancel)
- Clicking the checkbox toggles completion
- Clicking the edit button opens/closes the detail editor for that task
- Clicking "+" creates a new blank task below the selected task, or at the bottom if nothing is selected
- Typing in the search field filters the visible tasks by name
- Completed tasks appear with strikethrough styling
