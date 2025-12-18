# Interfaces

## Frontend Server

The webapp that runs in the browser is a custom "frontend server" that hosts the remote user interface controlled by the backend.

**Session URLs:**
- Connecting to `http://SITE` redirects to `http://SITE/NEW-SESSION-ID`
- Each session has a unique ID embedded in the URL path
- Session URLs can be shared or bookmarked for reconnection

**Shared Worker Architecture:**
- A SharedWorker maintains the concept of a "main" tab connected to the backend
- The main tab holds the primary WebSocket connection
- Other tabs coordinate through the SharedWorker
- Connecting to `http://SITE/SESSION-ID` when a tab is already connected:
  - Activates the existing connected tab (via desktop notification)
  - Closes the new tab

**SPA History Management:**
- Each session uses SPA-style history management
- Objects (presenters) can be registered with URL paths by the backend app (after the session ID)
  - Only presenters explicitly registered by the backend are addressable via URL
- Navigation updates the URL without full page reloads
- Back/forward navigation restores presenter state

**Tab Activation:**

Opening a new browser tab/window to a session URL:
- `http://SITE/SESSION-ID` - Activates the existing connected tab
  - Sends a "click to focus" desktop notification
  - Clicking the notification brings the main tab to focus
  - If history length == 1, closes the new tab; otherwise goes back in history
- `http://SITE/SESSION-ID/PATH` - Activates and navigates to PATH
  - Backend can open this URL to direct the user to a specific page
  - Same activation behavior as above
- `http://SITE/SESSION-ID` when no session exists - Shows an error page
  - No desktop notification is sent
  - The tab remains open (not auto-closed)

Enables backends/AIs to bring the UI to the user's attention by opening the session URL.

**Reconnection:**
- Frontend can reconnect to any session that hasn't timed out
- Session timeout is configured via `--session-timeout` (default: 24h, 0=never)
- Allows page refreshes, network interruptions, and browser restarts without losing session state
- Session state is preserved until session timeout expires

**Browser Communication:**
- **WebSocket**: Real-time bidirectional communication (via main tab)
- **JSONP**: For legacy/cross-origin scenarios

## Backend Integration Patterns

- **REST API (HTTP)**: Standard request/response
- **WebSocket**: Persistent real-time connection
- **MCP (Model Context Protocol)**: For AI assistant integration (see MCP Server below)
- **Command Line**: Mirrors the REST API for simple shell script integration
- **Embedded Lua**: Backend logic in the `lua/` subdirectory runs within the UI server process

## Backend Modes

The UI server supports three backend configurations:

**1. Embedded Lua only (`--lua`, no connected backend):**
- Complete app runs in embedded Lua
- `main.lua` creates variable 1 and handles all logic
- Best for: Simple apps, demos, prototypes

**2. Connected backend only (no `--lua`):**
- Complete app runs in external backend (Go, etc.) connected via socket
- Backend creates variable 1 and handles all logic
- Best for: Apps that need full backend language capabilities

**3. Hybrid (`--lua` + connected backend):**
- Both embedded Lua and external backend active
- Developer decides where variable 1 is created
- Allows embedded Lua for reusable UI behavior with backend as "plugin"
- Best for: Complex apps where Lua handles common patterns and backend handles app-specific logic

## Backend Socket

External backends connect to the UI server via Unix socket (or named pipe on Windows).

**Protocol:**
- Session-wrapped batches: `{"session": "abc123", "messages": [...]}`
- When a batch arrives with a new session ID, the backend creates a corresponding session
- Backend is responsible for creating variable 1 (unless hybrid mode with Lua creating it)

**Default path:** `/tmp/ui.sock` (Unix) or `\\.\pipe\ui` (Windows)

## Embedded Lua Runtime

When `--lua` is enabled (default: true), the UI server provides an embedded Lua runtime for presentation logic.

**Session-Based Architecture:**
- Each frontend session has a corresponding Lua session
- When a frontend connects and creates a new session, the UI server:
  1. Creates a new Lua session with a `session` global
  2. Loads and executes `main.lua` from the site's `lua/` directory (if it exists)
- Executing `main.lua` serves as the notification of a new session
- In Lua-only mode, `main.lua` is responsible for:
  - Creating variable 1 (the app variable) with initial app state
  - Defining presenter objects with methods that handle `ui-action` calls
- In hybrid mode, `main.lua` may set up reusable behaviors while the backend creates variable 1

**Session Object:**
- A `session` global is available when `main.lua` executes
- Provides methods for variable management (see Lua Session API in libraries.md)

**Execution Model:**
- UI server creates a Lua executor goroutine with a channel for zero-arg functions
- All variable path sets and method calls execute through this channel
- Ensures single-threaded Lua access (Lua VMs are not thread-safe)
- Variable updates that trigger Lua methods are queued and processed sequentially

**Dynamic Code Loading:**
- After initial load, additional code can be loaded via the `lua` property on variable 1
- Two modes depending on the value format:
  1. **Inline code**: If the value is Lua code, evaluate it directly
  2. **File reference**: If the value ends with `.lua`, load from `<dir>/lua/<filename>`

- Examples via protocol:
  ```json
  // Inline code
  {"type": "update", "id": 1, "properties": {"lua": "ui.registerPresenter('MyApp', {...})"}}

  // File reference
  {"type": "update", "id": 1, "properties": {"lua": "helpers.lua"}}
  ```

**Lua API:**
- `ui.registerPresenter(name, table)` - Register a presenter type
- `ui.log([level,] message)` - Log from Lua code (delegates to `Config.Log`)
- `ui.json_encode(value)` / `ui.json_decode(string)` - JSON conversion

## MCP Server

The UI platform provides an MCP server, enabling AIs to create and control user interfaces. This is the primary integration point for AI assistants.

**Architecture:**
- MCP connects as a backend to a single running UI server
- Presentation logic runs in embedded Lua within the UI server
- AI can load presenter logic (Lua code) via MCP tools
- AI interacts via MCP tools to create/update presenters and handle user input

**Two-Way Interaction:**
- UIs provide easy access to the AI for users, enabling smooth bidirectional communication
- Users can make spontaneous requests via UI widgets (e.g., chat widget, command input)
- AI receives user-initiated messages and can respond with UI updates
- Creates a natural conversation loop: AI presents → user interacts → AI responds

**Frictionless UI Creation:**
- AI can create new presenter types on-the-fly without prior registration
- AI generates viewdef HTML with `ui-*` bindings tailored to the data being presented
- No compile step or deployment required - create and display immediately
- Enables custom, intuitive presentations for any data structure or workflow
- AI can iterate on UI design in real-time based on user feedback

**MCP Resources:**
- List available presenter types and their properties
- List available viewdefs (TYPE.VIEW) and their bindings
- Query current session state
- Read pending user messages/requests

**MCP Tools:**
- Create and manage sessions
- Create, update, and destroy variables/presenters
- Create and update viewdefs (HTML templates with `ui-*` bindings)
- Load presenter logic (Lua code) into the UI server
- Register URL paths for presenters
- Activate user's browser tab
- Receive user input events and messages

**Workflow:**
1. AI queries available presenter types and viewdefs via MCP resources
2. AI calls MCP tool to create a session → receives session URL
3. AI opens the URL for the user (or shares it)
4. AI creates presenters and viewdefs via MCP tools
5. Lua presentation logic handles user interactions
6. User can send messages/requests to AI via chat widget or other inputs
7. AI receives events/messages and responds with UI updates or actions

**Usage Examples:**

*Example 1: Simple Data Display*
```
AI: create_session() → session_id: "abc123", url: "http://ui.local/abc123"
AI: create_viewdef("Report.DEFAULT", """
    <div>
      <h2 ui-value="title"></h2>
      <p ui-value="summary"></p>
      <div ui-content="body"></div>
    </div>
    """)
AI: create_presenter(type: "Report", properties: {
      title: "Q4 Analysis",
      summary: "Revenue up 15%",
      body: "<ul><li>Sales: $1.2M</li><li>Costs: $800K</li></ul>"
    })
→ User sees formatted report in browser
```

*Example 2: Interactive Form*
```
AI: create_viewdef("ContactForm.DEFAULT", """
    <sl-card>
      <div slot="header">Contact Us</div>
      <sl-input ui-value="name" label="Name"></sl-input>
      <sl-input ui-value="email" label="Email"></sl-input>
      <sl-textarea ui-value="message" label="Message"></sl-textarea>
      <sl-button ui-action="submit()">Send</sl-button>
    </sl-card>
    """)
AI: create_presenter(type: "ContactForm")
→ User fills form, clicks Send
AI: receives event {action: "submit", values: {name: "Alice", email: "...", message: "..."}}
AI: update_presenter(properties: {submitted: true})
```

*Example 3: Live Data Dashboard*
```
AI: load_presenter_logic("""
    function StockTicker:update(symbol, price, change)
      self.symbol = symbol
      self.price = price
      self.change = change
      self.changeClass = change >= 0 and "positive" or "negative"
    end
    """)
AI: create_viewdef("StockTicker.DEFAULT", """
    <div class="ticker">
      <span ui-value="symbol"></span>
      <span ui-value="price"></span>
      <span ui-value="change" ui-class-change="changeClass"></span>
    </div>
    """)
AI: create_presenter(type: "StockTicker")
AI: update_presenter(call: "update", args: ["ACME", 142.50, 3.25])
→ User sees live-updating stock ticker
```

*Example 4: Chat Interface with User Requests*
```
AI: create_viewdef("Chat.DEFAULT", """
    <div class="chat-container">
      <div ui-viewlist="messages" ui-namespace="ChatMessage"></div>
      <sl-input ui-value="input" placeholder="Ask me anything..."></sl-input>
      <sl-button ui-action="send()">Send</sl-button>
    </div>
    """)
AI: create_presenter(type: "Chat", properties: {messages: []})
→ User types "What's the weather?" and clicks Send
AI: receives event {action: "send", values: {input: "What's the weather?"}}
AI: update_presenter(properties: {
      messages: [..., {role: "user", text: "What's the weather?"}]
    })
AI: (fetches weather data)
AI: update_presenter(properties: {
      messages: [..., {role: "assistant", text: "Currently 72°F and sunny"}]
    })
```

*Example 5: Dynamic Table with Tabulator*
```
AI: create_viewdef("DataGrid.DEFAULT", """
    <div ui-tabulator="data"
         ui-columns='[{"field":"name","title":"Name"},{"field":"value","title":"Value"}]'>
    </div>
    """)
AI: create_presenter(type: "DataGrid", properties: {
      data: [
        {name: "Alpha", value: 100},
        {name: "Beta", value: 200}
      ]
    })
→ User sees sortable, filterable table
AI: update_presenter(properties: {data: [..., {name: "Gamma", value: 300}]})
→ Table updates with new row
```

*Example 6: Interactive Code Review with Claude*
```
→ User asks Claude to review a file
AI: create_viewdef("CodeReview.DEFAULT", """
    <div class="code-review">
      <div class="file-header">
        <span ui-value="filename"></span>
        <sl-badge ui-value="issueCount" variant="warning"></sl-badge>
      </div>
      <div ui-viewlist="annotations" ui-namespace="Annotation"></div>
      <div class="actions">
        <sl-button ui-action="applyFix()" variant="primary">Apply Selected Fixes</sl-button>
        <sl-button ui-action="explain()">Explain Issue</sl-button>
      </div>
    </div>
    """)
AI: create_viewdef("Annotation.DEFAULT", """
    <div class="annotation">
      <sl-checkbox ui-value="selected"></sl-checkbox>
      <code ui-value="lineRange"></code>
      <span ui-value="issue" ui-class-severity="severity"></span>
      <pre ui-content="suggestedFix"></pre>
    </div>
    """)
AI: create_presenter(type: "CodeReview", properties: {
      filename: "auth.go",
      issueCount: 3,
      annotations: [
        {lineRange: "42-45", issue: "SQL injection vulnerability", severity: "critical",
         suggestedFix: "db.Query(\"SELECT * FROM users WHERE id = ?\", userID)", selected: true},
        {lineRange: "78", issue: "Unused variable", severity: "warning",
         suggestedFix: "// Remove: var temp = ...", selected: false}
      ]
    })
→ User sees annotated code review, selects fixes
AI: receives event {action: "apply_fix"}
AI: (applies selected fixes to the actual file)
AI: update_presenter(properties: {issueCount: 1, annotations: [...]})
```

*Example 7: Live Test Runner Dashboard*
```
→ Claude runs tests and shows live results
AI: create_viewdef("TestRunner.DEFAULT", """
    <sl-card>
      <div slot="header">
        <span>Test Results</span>
        <sl-progress-bar ui-value="progress"></sl-progress-bar>
      </div>
      <div ui-viewlist="tests" ui-namespace="TestResult"></div>
      <div class="summary">
        <sl-badge variant="success" ui-value="passed"></sl-badge> passed
        <sl-badge variant="danger" ui-value="failed"></sl-badge> failed
      </div>
      <sl-button ui-action="rerunFailed()">Re-run Failed</sl-button>
      <sl-button ui-action="showCoverage()">Show Coverage</sl-button>
    </sl-card>
    """)
AI: create_viewdef("TestResult.DEFAULT", """
    <div class="test-result" ui-class-status="status">
      <sl-icon ui-attr-name="icon"></sl-icon>
      <span ui-value="name"></span>
      <span ui-value="duration"></span>
      <sl-button ui-action="viewDetails()" size="small">Details</sl-button>
    </div>
    """)
AI: create_presenter(type: "TestRunner", properties: {progress: 0, passed: 0, failed: 0, tests: []})
→ As tests run, Claude streams updates:
AI: update_presenter(properties: {
      progress: 25,
      passed: 5,
      tests: [..., {name: "TestAuth", status: "pass", duration: "0.3s", icon: "check-circle"}]
    })
AI: update_presenter(properties: {
      progress: 50,
      failed: 1,
      tests: [..., {name: "TestPayment", status: "fail", duration: "1.2s", icon: "x-circle"}]
    })
→ User clicks "view_details" on failed test
AI: receives event {action: "view_details", context: {name: "TestPayment"}}
AI: (creates detailed error view with stack trace)
```

*Example 8: Collaborative Debugging Session*
```
→ User asks Claude to help debug an issue
AI: create_viewdef("Debugger.DEFAULT", """
    <div class="debugger">
      <div class="source-panel">
        <pre ui-content="sourceCode"></pre>
        <div ui-viewlist="breakpoints" ui-namespace="Breakpoint"></div>
      </div>
      <div class="state-panel">
        <h4>Variables</h4>
        <div ui-tabulator="variables"
             ui-columns='[{"field":"name"},{"field":"value"},{"field":"type"}]'>
        </div>
        <h4>Call Stack</h4>
        <sl-tree ui-content="callStack"></sl-tree>
      </div>
      <div class="controls">
        <sl-button ui-action="stepOver()">Step Over</sl-button>
        <sl-button ui-action="stepInto()">Step Into</sl-button>
        <sl-button ui-action="continue()">Continue</sl-button>
        <sl-input ui-value="evalExpr" placeholder="Evaluate expression..."></sl-input>
        <sl-button ui-action="eval()">Eval</sl-button>
      </div>
      <div class="ai-insights">
        <h4>Claude's Analysis</h4>
        <div ui-content="analysis"></div>
      </div>
    </div>
    """)
AI: create_presenter(type: "Debugger", properties: {
      sourceCode: "<highlighted source with line numbers>",
      variables: [{name: "user", value: "nil", type: "*User"}],
      analysis: "<p>The null pointer occurs because <code>getUser()</code> returns nil when...</p>"
    })
→ User clicks "Eval" with expression "user.ID"
AI: receives event {action: "eval", values: {evalExpr: "user.ID"}}
AI: update_presenter(properties: {
      evalResult: "Error: nil pointer dereference",
      analysis: "<p>Confirmed: <code>user</code> is nil. The fix is to add a nil check on line 42...</p>"
    })
→ User can interact with Claude's analysis and apply suggested fixes
```

*Example 9: Architecture Explorer for Large Codebases*
```
→ User asks Claude to help understand a large project
AI: create_viewdef("ArchExplorer.DEFAULT", """
    <div class="arch-explorer">
      <div class="sidebar">
        <sl-input ui-value="search" placeholder="Search modules..."></sl-input>
        <sl-tree ui-content="moduleTree"></sl-tree>
      </div>
      <div class="main-view">
        <div class="diagram" ui-content="dependencyGraph"></div>
        <div class="details">
          <h3 ui-value="selectedModule"></h3>
          <p ui-value="description"></p>
          <div ui-tabulator="publicAPI"
               ui-columns='[{"field":"name"},{"field":"signature"},{"field":"usage"}]'>
          </div>
        </div>
      </div>
      <div class="metrics">
        <sl-badge ui-value="complexity">Complexity</sl-badge>
        <sl-badge ui-value="coupling">Coupling</sl-badge>
        <sl-badge ui-value="coverage">Coverage</sl-badge>
      </div>
    </div>
    """)
AI: create_presenter(type: "ArchExplorer", properties: {
      moduleTree: "<sl-tree-item>src/<sl-tree-item>auth/</sl-tree-item>...</sl-tree-item>",
      dependencyGraph: "<svg>...</svg>",  // Interactive dependency visualization
      selectedModule: "auth/",
      description: "Handles authentication, sessions, and API tokens",
      publicAPI: [
        {name: "Authenticate", signature: "(ctx, creds) -> (User, error)", usage: "47 refs"},
        {name: "ValidateToken", signature: "(token) -> (Claims, error)", usage: "123 refs"}
      ],
      complexity: "Medium", coupling: "Low", coverage: "78%"
    })
→ User clicks on a module in the tree
AI: receives event {selected: "payments/"}
AI: update_presenter(properties: {
      selectedModule: "payments/",
      dependencyGraph: "<svg>...updated graph highlighting payments...</svg>",
      publicAPI: [...payments API...]
    })
→ User can explore architecture interactively, Claude explains relationships
```

*Example 10: Multi-File Refactoring Planner*
```
→ User asks Claude to refactor a cross-cutting concern
AI: create_viewdef("RefactorPlan.DEFAULT", """
    <sl-card>
      <div slot="header">
        <h3 ui-value="refactorName"></h3>
        <sl-tag ui-value="impactLevel" ui-attr-variant="impactVariant"></sl-tag>
      </div>
      <p ui-content="summary"></p>
      <sl-divider></sl-divider>
      <h4>Affected Files (<span ui-value="fileCount"></span>)</h4>
      <div ui-viewlist="changes" ui-namespace="FileChange"></div>
      <sl-divider></sl-divider>
      <div class="actions">
        <sl-button ui-action="previewAll()">Preview All Changes</sl-button>
        <sl-button ui-action="applyAll()" variant="primary">Apply All</sl-button>
        <sl-button ui-action="applySelected()">Apply Selected</sl-button>
        <sl-button ui-action="generatePR()">Generate PR</sl-button>
      </div>
    </sl-card>
    """)
AI: create_viewdef("FileChange.DEFAULT", """
    <sl-details>
      <div slot="summary">
        <sl-checkbox ui-value="selected"></sl-checkbox>
        <code ui-value="path"></code>
        <sl-badge ui-value="changeType"></sl-badge>
        <span ui-value="lineChanges"></span>
      </div>
      <div class="diff" ui-content="diff"></div>
      <sl-button ui-action="editChange()" size="small">Edit</sl-button>
    </sl-details>
    """)
AI: create_presenter(type: "RefactorPlan", properties: {
      refactorName: "Extract Logger Interface",
      impactLevel: "Medium", impactVariant: "warning",
      summary: "Replace direct log calls with injectable Logger interface across 23 files...",
      fileCount: 23,
      changes: [
        {path: "pkg/logger/interface.go", changeType: "new", lineChanges: "+45",
         diff: "<pre>+ type Logger interface {...}</pre>", selected: true},
        {path: "cmd/server/main.go", changeType: "modify", lineChanges: "+3/-5",
         diff: "<pre>- log.Printf(...)\n+ logger.Info(...)</pre>", selected: true},
        ...
      ]
    })
→ User reviews changes, deselects some, clicks "Apply Selected"
AI: receives event {action: "apply_selected"}
AI: (applies changes to selected files)
AI: update_presenter(properties: {fileCount: 5, changes: [remaining...]})
→ User clicks "Generate PR"
AI: receives event {action: "generate_pr"}
AI: (creates branch, commits, opens PR)
AI: update_presenter(properties: {prUrl: "https://github.com/..."})
```

*Example 11: Dependency Upgrade Assistant*
```
→ User asks Claude to help upgrade dependencies
AI: create_viewdef("DepUpgrade.DEFAULT", """
    <div class="dep-upgrade">
      <div class="filters">
        <sl-select ui-value="filter" ui-items="filterOptions">
          <sl-option value="all">All</sl-option>
          <sl-option value="security">Security Updates</sl-option>
          <sl-option value="major">Major Versions</sl-option>
          <sl-option value="breaking">Breaking Changes</sl-option>
        </sl-select>
      </div>
      <div ui-tabulator="dependencies"
           ui-columns='[
             {"field":"select","formatter":"tickCross"},
             {"field":"name","title":"Package"},
             {"field":"current","title":"Current"},
             {"field":"latest","title":"Latest"},
             {"field":"type","title":"Type"},
             {"field":"risk","title":"Risk"}
           ]'>
      </div>
      <div class="analysis" ui-content="breakingChanges"></div>
      <div class="actions">
        <sl-button ui-action="analyzeSelected()">Analyze Impact</sl-button>
        <sl-button ui-action="upgradeSelected()" variant="primary">Upgrade Selected</sl-button>
        <sl-button ui-action="runTests()">Run Tests</sl-button>
      </div>
    </div>
    """)
AI: create_presenter(type: "DepUpgrade", properties: {
      dependencies: [
        {select: false, name: "react", current: "17.0.2", latest: "18.2.0",
         type: "major", risk: "medium"},
        {select: true, name: "lodash", current: "4.17.19", latest: "4.17.21",
         type: "patch", risk: "low"},
        {select: true, name: "axios", current: "0.21.1", latest: "1.6.0",
         type: "major", risk: "high"}
      ],
      breakingChanges: ""
    })
→ User selects axios, clicks "Analyze Impact"
AI: receives event {action: "analyze_selected"}
AI: (analyzes codebase for axios usage patterns)
AI: update_presenter(properties: {
      breakingChanges: """
        <h4>axios 0.21 → 1.6 Breaking Changes</h4>
        <ul>
          <li><strong>12 files affected</strong></li>
          <li>Response interceptors now receive AxiosError instead of Error</li>
          <li>Default timeout changed from 0 to 10000ms</li>
        </ul>
        <p>Recommended: Update error handling in src/api/*.ts</p>
      """
    })
→ User clicks "Upgrade Selected", Claude applies changes and updates impacted code
```

*Example 12: Feature Flag & Gradual Rollout Manager*
```
→ Claude helps manage feature development across a large team
AI: create_viewdef("FeatureManager.DEFAULT", """
    <div class="feature-manager">
      <div class="feature-list">
        <sl-input ui-value="search" placeholder="Search features..."></sl-input>
        <div ui-viewlist="features" ui-namespace="FeatureCard"></div>
        <sl-button ui-action="createFeature()">+ New Feature</sl-button>
      </div>
      <div class="feature-detail" ui-view="selectedFeature" ui-namespace="FeatureDetail"></div>
    </div>
    """)
AI: create_viewdef("FeatureCard.DEFAULT", """
    <sl-card class="feature-card" ui-class-status="status">
      <div slot="header">
        <span ui-value="name"></span>
        <sl-switch ui-value="enabled"></sl-switch>
      </div>
      <sl-progress-bar ui-value="rolloutPercent"></sl-progress-bar>
      <div class="stats">
        <span ui-value="branchCount"></span> branches
        <span ui-value="prCount"></span> PRs
      </div>
    </sl-card>
    """)
AI: create_viewdef("FeatureDetail.DEFAULT", """
    <div class="feature-detail">
      <h2 ui-value="name"></h2>
      <sl-textarea ui-value="description" label="Description"></sl-textarea>
      <h4>Implementation Status</h4>
      <div ui-viewlist="tasks" ui-namespace="FeatureTask"></div>
      <h4>Rollout Configuration</h4>
      <sl-range ui-value="rolloutPercent" min="0" max="100"></sl-range>
      <sl-select ui-value="targetEnv">
        <sl-option value="dev">Development</sl-option>
        <sl-option value="staging">Staging</sl-option>
        <sl-option value="prod">Production</sl-option>
      </sl-select>
      <h4>Code Locations</h4>
      <div ui-tabulator="codeRefs"
           ui-columns='[{"field":"file"},{"field":"line"},{"field":"type"}]'>
      </div>
      <sl-button ui-action="findUsages()">Find All Usages</sl-button>
      <sl-button ui-action="cleanupFlag()" variant="danger">Remove Flag (Ship It)</sl-button>
    </div>
    """)
AI: create_presenter(type: "FeatureManager", properties: {
      features: [
        {name: "new-checkout-flow", status: "active", enabled: true,
         rolloutPercent: 25, branchCount: 3, prCount: 2},
        {name: "ai-recommendations", status: "development", enabled: false,
         rolloutPercent: 0, branchCount: 1, prCount: 5}
      ]
    })
→ User clicks "Remove Flag (Ship It)" after 100% rollout
AI: receives event {action: "cleanup_flag", feature: "new-checkout-flow"}
AI: (finds all flag checks, removes conditionals, keeps feature code)
AI: update_presenter(properties: {
      cleanupPlan: {filesAffected: 8, linesRemoved: 47, linesKept: 234}
    })
→ Shows diff of cleanup changes for review before applying
```
