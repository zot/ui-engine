# App Shell

**Source**: interfaces.md, libraries.md
**Route**: /{sessionId} (see manifest-ui.md)

**Purpose**: Root view container that displays currentPage() presenter

---

**Data** (see crc-AppPresenter.md):
- `currentPage(): Presenter` - Active page from history[historyIndex]
- `url: string` - Current URL path
- `historyIndex: number` - Position in history stack
- `history: Presenter[]` - Page history array

---

**Layout**:
```
+--------------------------------------------------+
|                   App Shell                       |
|  +----------------------------------------------+|
|  |                                              ||
|  |           [currentPage() view]               ||
|  |                                              ||
|  |    Rendered using TYPE.VIEW viewdef          ||
|  |    where TYPE = currentPage().type           ||
|  |                                              ||
|  +----------------------------------------------+|
+--------------------------------------------------+
```

---

**Bindings**:
- Root element has `ui-view="currentPage()" ui-namespace="DEFAULT"`
- Nested views resolve via ui-namespace for TYPE.VIEW lookup

---

**Events** (see crc-SPANavigator.md):
- Browser `popstate` triggers `handlePopState()`
- Link navigation triggers URL resolution via Router

---

**CSS Classes**:
- `app-shell` - Root container
- Content styling defined by individual viewdefs

---

**Notes**:
- This is the minimal app shell; actual UI defined by viewdefs
- Backend controls all visual presentation via viewdefs
- SharedWorker handles multi-tab coordination invisibly
