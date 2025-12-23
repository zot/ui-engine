# MCP Interaction Guide for AI Agents

As an AI agent, you can use this platform to build "tiny apps" that facilitate rich collaboration with the user.

## Agent Workflow

### 1. Environment Setup
Always start by calling `ui_configure` with a base directory. This ensures log files are created and the filesystem is ready.

### 2. Startup
Call `ui_start` to activate the HTTP server. It will return a URL. You can then use `ui_open_browser` to show the UI to the user.

### 3. Dynamic UI Creation
You don't need pre-existing files. You can:
1. Upload viewdefs for your types via `ui_upload_viewdef`.
2. Define your logic via `ui_run`.
3. Instantiate your logic and attach it to the screen.

### 4. Collaborative Loop
1. **Show a form:** Upload a viewdef with an input and a button.
2. **Handle input:** Use a method in Lua that updates state or sends a notification.
3. **Wait for user:** Your `ui_run` finishes, but the server stays running. When the user clicks the button, your Lua code executes.
4. **Receive Notification:** If your Lua code calls `mcp.notify`, you will receive a notification.
5. **Inspect State:** Use `ui://state` to see the results of the user's interaction.
6. **Update UI:** Upload a new viewdef or run more Lua code to show the next step.

## Best Practices

- **Atomic Viewdefs:** Keep viewdefs small and focused on a single type.
- **Clear Logic:** Use the `mcp.state` to expose only what is relevant to your current task.
- **Informative Notifications:** Use `mcp.notify` to tell yourself when a meaningful user action has occurred.
- **Log Inspection:** Check the `{base_dir}/log/lua.log` if things aren't working as expected. You can use standard filesystem tools or a hypothetical `ui_read_logs` tool (if implemented).

## Example: A Quick Feedback Form

```lua
-- Lua code
MyFeedback = { type = "MyFeedback" }
MyFeedback.__index = MyFeedback
function MyFeedback:submit()
    mcp.notify("feedback_received", { rating = self.rating })
end

feedbackVal = MyFeedback:new({ rating = 5 })
mcp.state = feedbackVal
```

```html
<!-- HTML viewdef -->
<template>
  <div class="feedback">
    <h3>How am I doing?</h3>
    <sl-rating ui-value="rating"></sl-rating>
    <sl-button ui-action="submit()">Send Feedback</sl-button>
  </div>
</template>
```
