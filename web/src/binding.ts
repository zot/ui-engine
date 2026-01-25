// Binding engine for ui-* attributes
// CRC: crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md, crc-HtmlBinding.md
// Spec: viewdefs.md

import { VariableStore, VariableError } from './connection'
import { ensureElementId, vendElementId } from './element_id_vendor'

const READ_ONLY_WITH_VALUE = {
  'SL-BADGE': true
}

// Spec: viewdefs.md - Shoelace components with ui-value support that are read-only
// These components have a value property but no user-editable input
// CRC: crc-ValueBinding.md - Read-only Shoelace components
const READONLY_SHOELACE_TAGS = new Set([
  'SL-COPY-BUTTON',
  'SL-OPTION',
  'SL-PROGRESS-BAR',
  'SL-PROGRESS-RING',
  'SL-QR-CODE',
])

// Elements that don't resize when their content changes
// These should NOT trigger parent scroll notifications on ui-value updates
// CRC: crc-ValueBinding.md - Parent Scroll Notifications
const NON_RESIZING_ELEMENTS = new Set([
  'INPUT', 'TEXTAREA', 'SL-INPUT', 'SL-TEXTAREA'
])

// Check if element triggers parent scroll notifications on content update
// Content-resizable elements (span, div, etc.) trigger scroll, input elements don't
// CRC: crc-ValueBinding.md - Parent Scroll Notifications
function triggersParentScroll(element: Element): boolean {
  return !NON_RESIZING_ELEMENTS.has(element.tagName)
}

export interface PathOptions {
  create?: string
  wrapper?: string
  item?: string
  props?: Record<string, string>
}

export interface ParsedPath {
  segments: string[]
  options: PathOptions
}

// Spec: viewdefs.md - Elements with tag names starting with sl- default to read-write
function isShoelaceComponent(element: any) {
  return element instanceof HTMLElement && element.nodeName.startsWith('SL-')
}

// Parse a path like "father.name?create=Person&wrapper=lua.ViewList&itemWrapper=ContactPresenter"
// Properties without values default to "true": "name?keypress" equals "name?keypress=true"
// Spec: protocol.md - Path property syntax, libraries.md - View rendering
export function parsePath(path: string): ParsedPath {
  const [pathPart, queryPart] = path.split('?')
  const segments = pathPart.split('.')
  const options: PathOptions = {}

  if (queryPart) {
    const params = new URLSearchParams(queryPart)

    // Helper to get value, defaulting empty to "true"
    const getValue = (key: string): string => {
      const val = params.get(key)
      return val === '' ? 'true' : val!
    }

    // Extract well-known properties
    if (params.has('create')) {
      options.create = getValue('create')
    }
    if (params.has('wrapper')) {
      options.wrapper = getValue('wrapper')
    }
    if (params.has('item')) {
      options.item = getValue('item')
    }

    // Collect remaining properties (empty values become "true")
    const props: Record<string, string> = {}
    params.forEach((value, key) => {
      if (key !== 'create' && key !== 'wrapper' && key !== 'item') {
        props[key] = value === '' ? 'true' : value
      }
    })
    if (Object.keys(props).length > 0) {
      options.props = props
    }
  }

  return { segments, options }
}

// Convert path options to variable properties map
// Used when creating variables from paths with properties
export function pathOptionsToProperties(
  options: PathOptions
): Record<string, string> {
  const props: Record<string, string> = {}

  if (options.create) {
    props['create'] = options.create
  }
  if (options.wrapper) {
    props['wrapper'] = options.wrapper
  }
  if (options.item) {
    props['item'] = options.item
  }
  if (options.props) {
    Object.assign(props, options.props)
  }

  return props
}

// Widget: Binding context for an element with ui-* bindings
// CRC: crc-Widget.md
// Spec: viewdefs.md - Widgets
// Forward declaration for View type (avoid circular import)
export interface ViewLike {
  forceRender(): void;
}

export class Widget {
  readonly elementId: string
  private variables: Map<string, number> = new Map()  // binding name → variable ID
  private unbindHandlers: Map<string, () => void> = new Map()  // binding name → cleanup fn
  view?: ViewLike  // Optional reference to containing View (for hot-reload)
  scrollOnOutput = false  // CRC: crc-Widget.md - scrollOnOutput property

  constructor(element: Element) {
    // Vend ID if element doesn't have one
    if (!element.id) {
      element.id = ensureElementId(element)
    }
    this.elementId = element.id
  }

  // Scroll element to bottom if scrollable
  // CRC: crc-Widget.md - scrollToBottom
  scrollToBottom(): void {
    const element = this.getElement()
    if (element && element.scrollHeight > element.clientHeight) {
      element.scrollTop = element.scrollHeight
    }
  }

  // Register a binding's variable ID and unbind handler
  registerBinding(name: string, varId: number, unbindHandler: () => void): void {
    this.variables.set(name, varId)
    this.unbindHandlers.set(name, unbindHandler)
  }

  // Get variable ID for a binding name
  getVariableId(name: string): number | undefined {
    return this.variables.get(name)
  }

  // Check if a binding exists
  hasBinding(name: string): boolean {
    return this.variables.has(name)
  }

  // Get the DOM element
  getElement(): Element | null {
    return document.getElementById(this.elementId)
  }

  // Call all unbind handlers and clean up
  unbindAll(): void {
    for (const handler of this.unbindHandlers.values()) {
      handler()
    }
    this.unbindHandlers.clear()
    this.variables.clear()
  }
}

// Resolve a path against a variable value
export function resolvePath(value: unknown, segments: string[]): unknown {
  let current = value
  for (const segment of segments) {
    if (current === null || current === undefined) {
      return undefined
    }
    if (typeof current === 'object') {
      // Handle array index
      if (Array.isArray(current) && /^\d+$/.test(segment)) {
        current = current[parseInt(segment, 10)]
      } else {
        current = (current as Record<string, unknown>)[segment]
      }
    } else {
      return undefined
    }
  }
  return current
}

export class BindingEngine {
  private store: VariableStore
  private widgets: Map<string, Widget> = new Map()  // keyed by elementId
  private pendingScrollNotifications: Set<number> = new Set()  // variable IDs to notify
  private scrollProcessingScheduled = false  // whether processing is already queued

  constructor(store: VariableStore) {
    this.store = store
  }

  // Add a variable ID to pending scroll notifications
  // Called by Views after rendering to notify parent that child rendered
  // Schedules processing via queueMicrotask to batch notifications
  // Spec: viewdefs.md - Render notifications (for scrollOnOutput)
  // CRC: crc-BindingEngine.md - addScrollNotification
  addScrollNotification(varId: number): void {
    this.pendingScrollNotifications.add(varId)

    // Schedule processing if not already scheduled
    if (!this.scrollProcessingScheduled) {
      this.scrollProcessingScheduled = true
      queueMicrotask(() => {
        this.scrollProcessingScheduled = false
        this.processScrollNotifications()
      })
    }
  }

  // Process pending scroll notifications after batch completes
  // Uses current/next pattern to bubble up until a widget with scrollOnOutput is found
  // Spec: viewdefs.md - Render notifications (for scrollOnOutput)
  // CRC: crc-BindingEngine.md - processScrollNotifications
  processScrollNotifications(): void {
    let current = new Set(this.pendingScrollNotifications)
    this.pendingScrollNotifications.clear()

    while (current.size > 0) {
      const next = new Set<number>()

      for (const varId of current) {
        const varData = this.store.get(varId)
        if (!varData) continue

        const elementId = varData.properties['elementId']
        if (elementId) {
          const widget = this.widgets.get(elementId)
          // Check if widget has scrollOnOutput (universal property on widget, not view)
          // CRC: crc-Widget.md - scrollOnOutput property
          if (widget?.scrollOnOutput) {
            // Scroll and don't bubble further
            widget.scrollToBottom()
            continue
          }
        }

        // Bubble up to parent
        if (varData.parentId) {
          next.add(varData.parentId)
        }
      }

      current = next
    }
  }

  // Get widget for an element (for external access if needed)
  getWidget(elementId: string): Widget | undefined {
    return this.widgets.get(elementId)
  }

  // Get view for an element (via widget.view, for hot-reload)
  // Spec: viewdefs.md - Hot-reload re-rendering
  getView(elementId: string): ViewLike | undefined {
    const widget = this.widgets.get(elementId)
    return widget?.view
  }

  // Set view for a widget (creating widget if needed)
  // Used by View to register itself for hot-reload
  // Spec: viewdefs.md - Hot-reload re-rendering
  setViewForElement(elementId: string, view: ViewLike): void {
    let widget = this.widgets.get(elementId)
    if (!widget) {
      const element = document.getElementById(elementId)
      if (!element) return
      widget = new Widget(element)
      this.widgets.set(elementId, widget)
    }
    widget.view = view
  }

  // Bind all ui-* attributes on an element
  // Widget owns all bindings via unbindHandlers
  bindElement(element: Element, contextVarId: number): void {
    let hasBindings = false

    // Create Widget for this element's bindings
    // CRC: crc-Widget.md
    const widget = new Widget(element)

    // ui-value binding (processed first so event bindings can reference it)
    const uiValue = element.getAttribute('ui-value')
    if (uiValue) {
      this.createValueBinding(element, contextVarId, uiValue, widget)
      hasBindings = true
    }

    // ui-attr-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-attr-')) {
        const targetAttr = attr.name.substring(8) // Remove "ui-attr-"
        this.createAttrBinding(contextVarId, attr.value, targetAttr, widget)
        hasBindings = true
      }
    }

    // ui-class-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-class-')) {
        const className = attr.name.substring(9) // Remove "ui-class-"
        this.createClassBinding(contextVarId, attr.value, className, widget)
        hasBindings = true
      }
    }

    // ui-style-*-* bindings (e.g., ui-style-background-color)
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-style-')) {
        const styleProp = attr.name.substring(9) // Remove "ui-style-"
        this.createStyleBinding(contextVarId, attr.value, styleProp, widget)
        hasBindings = true
      }
    }

    // ui-event-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-event-')) {
        const eventName = attr.name.substring(9) // Remove "ui-event-"
        this.createEventBinding(element, contextVarId, attr.value, eventName, widget)
        hasBindings = true
      }
    }

    // ui-action binding (shorthand for click action)
    const uiAction = element.getAttribute('ui-action')
    if (uiAction) {
      this.createActionBinding(element, contextVarId, uiAction, widget)
      hasBindings = true
    }

    // ui-code binding (execute JS when variable updates)
    const uiCode = element.getAttribute('ui-code')
    if (uiCode) {
      this.createCodeBinding(contextVarId, uiCode, widget)
      hasBindings = true
    }

    // ui-html binding (set innerHTML or replace element)
    const uiHtml = element.getAttribute('ui-html')
    if (uiHtml) {
      this.createHtmlBinding(contextVarId, uiHtml, widget)
      hasBindings = true
    }

    if (hasBindings) {
      this.widgets.set(widget.elementId, widget)
    }

    // Recursively bind children
    for (const child of Array.from(element.children)) {
      this.bindElement(child, contextVarId)
    }
  }

  // Unbind all bindings from an element and its children
  unbindElement(element: Element): void {
    const elementId = element.id
    if (elementId) {
      const widget = this.widgets.get(elementId)
      if (widget) {
        widget.unbindAll()
        this.widgets.delete(elementId)
      }
    }

    for (const child of Array.from(element.children)) {
      this.unbindElement(child)
    }
  }

  // Create a value binding (sets textContent or value, and handles changes)
  // Spec: viewdefs.md - Nullish path handling with error indicators
  // Spec: libraries.md - Input update behavior (blur by default, keypress for immediate)
  private createValueBinding(
    element: Element,
    varId: number,
    path: string,
    widget: Widget
  ): void {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    // Check if keypress mode is enabled (send updates on every keypress vs blur)
    const useKeypress = parsed.options.props?.['keypress'] === 'true'

    // Check if scrollOnOutput is enabled (auto-scroll to bottom on value updates)
    // Spec: viewdefs.md - scrollOnOutput path property (universal property)
    // CRC: crc-Widget.md - scrollOnOutput property
    const scrollOnOutput = parsed.options.props?.['scrollOnOutput'] === 'true'
    if (scrollOnOutput) {
      widget.scrollOnOutput = true  // Set on widget for bubbling mechanism
    }

    // Create a child variable for this path
    // The server will resolve the path and send back the value
    let childVarId: number | null = null
    let unbindValue: (() => void) | null = null
    let unbindError: (() => void) | null = null
    // Custom elements (tagNames with hyphens) may not be upgraded yet when binding runs,
    // so 'value' in element would return false. Assume custom elements have value property.
    // Exclude buttons - they have a value property for forms but we want textContent.
    //const isCustomElement = element.tagName.includes('-')
    const isButton = element instanceof HTMLButtonElement
    const editableValue =
      !isButton &&
      (element instanceof HTMLInputElement ||
        element instanceof HTMLTextAreaElement ||
        element instanceof HTMLSelectElement ||
        isShoelaceComponent(element) ||
        ('value' in element && !(element.nodeName in READ_ONLY_WITH_VALUE)))

    // Capture element ID for closures to avoid holding DOM references
    // Spec: viewdefs.md - Element References (use ID lookup, not direct references)
    const elementId = widget.elementId

    // Helper to scroll element to bottom if scrollable
    const scrollToBottom = () => {
      if (scrollOnOutput) {
        const el = document.getElementById(elementId)
        if (el && el.scrollHeight > el.clientHeight) {
          el.scrollTop = el.scrollHeight
        }
      }
    }

    // Check if element is content-resizable (triggers parent scroll on value update)
    // CRC: crc-ValueBinding.md - Parent Scroll Notifications
    const shouldNotifyParentScroll = triggersParentScroll(element)

    const update = editableValue
      ? (value: unknown) => {
          const el = document.getElementById(elementId)
          if (!el) return
          // Preserve number type for components like sl-rating, sl-range
          if (typeof value === 'number') {
            (el as any).value = value
          } else if (value === null || value === undefined || value === '') {
            // sl-select: set empty string and clear displayLabel after component updates
            if (el.tagName.toLowerCase() === 'sl-select') {
              (el as any).value = ''
              // Clear displayLabel after component's update cycle
              setTimeout(() => {
                (el as any).displayLabel = ''
              }, 0)
            } else {
              (el as any).value = ''
            }
          } else {
            (el as any).value = value.toString()
          }
          scrollToBottom()
          // Content-resizable elements notify parent for scrollOnOutput
          // Input elements don't resize so they don't trigger parent scroll
          if (shouldNotifyParentScroll) {
            this.addScrollNotification(varId)
          }
        }
      : (value: unknown) => {
          const el = document.getElementById(elementId)
          if (!el) return
          el.textContent = value?.toString() ?? ''
          scrollToBottom()
          // Content-resizable elements notify parent for scrollOnOutput
          if (shouldNotifyParentScroll) {
            this.addScrollNotification(varId)
          }
        }

    // Handle error state changes - add/remove ui-error class and ui-error-* attributes
    const updateError = (error: VariableError | null) => {
      const el = document.getElementById(elementId)
      if (!el) return
      if (error) {
        el.classList.add('ui-error')
        el.setAttribute('ui-error-code', error.code)
        el.setAttribute('ui-error-description', error.description)
      } else {
        el.classList.remove('ui-error')
        el.removeAttribute('ui-error-code')
        el.removeAttribute('ui-error-description')
      }
    }

    // Two-way binding: listen for input changes
    const inputHandler = (e: Event) => {
      if (childVarId !== null) {
        const target = e.target as
          | HTMLInputElement
          | HTMLTextAreaElement
          | HTMLSelectElement
        this.store.update(childVarId, target.value)
      }
    }

    // Method calls (paths ending with ()) require access 'r' or 'action'
    const isMethodCall = parsed.segments[parsed.segments.length - 1]?.endsWith('()')
    if (isMethodCall) {
      if (properties['access'] === undefined) {
        properties.access = 'r'
      } else if (properties['access'] !== 'r' && properties['access'] !== 'action') {
        console.error(`Invalid access '${properties['access']}' for method call path '${path}' - must be 'r' or 'action'`)
        return
      }
    } else if (!editableValue && properties['access'] === undefined) {
      properties.access = 'r'
    } else if (READONLY_SHOELACE_TAGS.has(element.nodeName) && properties['access'] === undefined) {
      // Spec: viewdefs.md - Read-only Shoelace components default to access=r
      properties.access = 'r'
    }

    // Create the child variable asynchronously
    this.store
      .create({
        parentId: varId,
        properties,
        widget,  // CRC: crc-Variable.md - widget reference
      })
      .then((id) => {
        childVarId = id
        // Watch the child variable for value updates
        unbindValue = this.store.watch(id, (_v, value) => update(value))
        unbindError = this.store.watchErrors(id, updateError)

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) {
          update(current.value)
        }

        // Register with Widget - includes unbind handler
        // CRC: crc-Widget.md - registerBinding
        widget.registerBinding('ui-value', id, () => {
          if (unbindValue) unbindValue()
          if (unbindError) unbindError()
          if (nativeEventType) element.removeEventListener(nativeEventType, inputHandler)
          if (shoelaceEventType) element.removeEventListener(shoelaceEventType, shoelaceHandler)
          element.removeEventListener('ui-value-change', changeHandler)
          this.store.destroy(id)
          element.classList.remove('ui-error')
          element.removeAttribute('ui-error-code')
          element.removeAttribute('ui-error-description')
        })
      })
      .catch((err) => {
        console.error('Failed to create binding variable:', err)
      })

    // Determine which events to listen for based on element type and keypress setting
    // Native inputs: use 'input' for keypress mode, 'blur' for default
    // Shoelace inputs: use 'sl-input' for keypress mode, 'sl-change' for default
    const isNativeInput =
      element instanceof HTMLInputElement ||
      element instanceof HTMLTextAreaElement ||
      element instanceof HTMLSelectElement
    const tagLower = element.tagName.toLowerCase()
    const isShoelaceInput =
      tagLower === 'sl-input' || tagLower === 'sl-textarea' || tagLower === 'sl-select'

    let nativeEventType: string | null = null
    let shoelaceEventType: string | null = null

    if (isNativeInput) {
      nativeEventType = useKeypress ? 'input' : 'blur'
    }
    if (isShoelaceInput) {
      shoelaceEventType = useKeypress ? 'sl-input' : 'sl-change'
    }

    // Add native input listener
    if (nativeEventType) {
      element.addEventListener(nativeEventType, inputHandler)
    }

    // Add Shoelace event listener
    const shoelaceHandler = (e: Event) => {
      if (childVarId !== null) {
        const target = e.target as HTMLInputElement
        this.store.update(childVarId, target.value)
      }
    }
    if (shoelaceEventType) {
      element.addEventListener(shoelaceEventType, shoelaceHandler)
    }

    // Also listen for ui-value-change events from other custom widgets
    const changeHandler = (e: Event) => {
      const customEvent = e as CustomEvent
      if (childVarId !== null) {
        this.store.update(childVarId, customEvent.detail.value)
      }
    }
    element.addEventListener('ui-value-change', changeHandler)
  }

  // Create an attribute binding
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private createAttrBinding(
    varId: number,
    path: string,
    targetAttr: string,
    widget: Widget
  ): void {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    // Set scrollOnOutput on widget if specified (universal property)
    // CRC: crc-Widget.md - scrollOnOutput property
    if (parsed.options.props?.['scrollOnOutput'] === 'true') {
      widget.scrollOnOutput = true
    }

    // Default to access=r for attribute bindings (read-only)
    // Spec: viewdefs.md - Value Bindings
    if (!properties['access']) {
      properties['access'] = 'r'
    }

    // Capture element ID for closures to avoid holding DOM references
    const elementId = widget.elementId

    const update = (value: unknown) => {
      const el = document.getElementById(elementId)
      if (!el) return
      if (value !== null && value !== undefined && value !== false) {
        el.setAttribute(targetAttr, value.toString())
      } else {
        el.removeAttribute(targetAttr)
      }
    }

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
        widget,  // CRC: crc-Variable.md - widget reference
      })
      .then((id) => {
        const unbindValue = this.store.watch(id, (_v, value) => update(value))

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) update(current.value)

        // Register with Widget
        widget.registerBinding(`ui-attr-${targetAttr}`, id, () => {
          unbindValue()
          this.store.destroy(id)
        })
      })
      .catch((err) => {
        console.error('Failed to create attr binding variable:', err)
      })
  }

  // Create a class binding
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private createClassBinding(
    varId: number,
    path: string,
    className: string,
    widget: Widget
  ): void {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    // Set scrollOnOutput on widget if specified (universal property)
    // CRC: crc-Widget.md - scrollOnOutput property
    if (parsed.options.props?.['scrollOnOutput'] === 'true') {
      widget.scrollOnOutput = true
    }

    // Default to access=r for class bindings (read-only)
    // Spec: viewdefs.md - Value Bindings
    if (!properties['access']) {
      properties['access'] = 'r'
    }

    // Capture element ID for closures to avoid holding DOM references
    const elementId = widget.elementId

    const update = (value: unknown) => {
      const el = document.getElementById(elementId)
      if (!el) return
      if (value) {
        el.classList.add(className)
      } else {
        el.classList.remove(className)
      }
    }

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
        widget,  // CRC: crc-Variable.md - widget reference
      })
      .then((id) => {
        const unbindValue = this.store.watch(id, (_v, value) => update(value))

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) update(current.value)

        // Register with Widget
        widget.registerBinding(`ui-class-${className}`, id, () => {
          unbindValue()
          this.store.destroy(id)
        })
      })
      .catch((err) => {
        console.error('Failed to create class binding variable:', err)
      })
  }

  // Create a style binding
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private createStyleBinding(
    varId: number,
    path: string,
    styleProp: string,
    widget: Widget
  ): void {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    // Set scrollOnOutput on widget if specified (universal property)
    // CRC: crc-Widget.md - scrollOnOutput property
    if (parsed.options.props?.['scrollOnOutput'] === 'true') {
      widget.scrollOnOutput = true
    }

    // Default to access=r for style bindings (read-only)
    // Spec: viewdefs.md - Value Bindings
    if (!properties['access']) {
      properties['access'] = 'r'
    }

    // Capture element ID for closures to avoid holding DOM references
    const elementId = widget.elementId

    const update = (value: unknown) => {
      const el = document.getElementById(elementId) as HTMLElement | null
      if (!el) return
      if (value !== null && value !== undefined) {
        el.style.setProperty(styleProp, value.toString())
      } else {
        el.style.removeProperty(styleProp)
      }
    }

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
        widget,  // CRC: crc-Variable.md - widget reference
      })
      .then((id) => {
        const unbindValue = this.store.watch(id, (_v, value) => update(value))

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) update(current.value)

        // Register with Widget
        widget.registerBinding(`ui-style-${styleProp}`, id, () => {
          unbindValue()
          this.store.destroy(id)
        })
      })
      .catch((err) => {
        console.error('Failed to create style binding variable:', err)
      })
  }

  // Create a code binding (execute JS when variable updates)
  // Spec: viewdefs.md - Value Bindings (ui-code)
  private createCodeBinding(
    varId: number,
    path: string,
    widget: Widget
  ): void {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    // Set scrollOnOutput on widget if specified (universal property)
    // CRC: crc-Widget.md - scrollOnOutput property
    if (parsed.options.props?.['scrollOnOutput'] === 'true') {
      widget.scrollOnOutput = true
    }

    // Default to access=r for code bindings (read-only)
    // Spec: viewdefs.md - Value Bindings
    if (!properties['access']) {
      properties['access'] = 'r'
    }

    // Capture element ID for closures to avoid holding DOM references
    const elementId = widget.elementId

    let childVarId: number | null = null

    // Execute code with element, value, variable, and store in scope
    // Spec: viewdefs.md - ui-code binding scope
    const executeCode = (code: unknown) => {
      if (typeof code !== 'string' || !code) return
      const el = document.getElementById(elementId)
      if (!el) return
      try {
        // Create function with element, value, variable, and store parameters
        const fn = new Function('element', 'value', 'variable', 'store', code)
        // Get current variable data from store
        const current = childVarId !== null ? this.store.get(childVarId) : null
        fn(el, current?.value, current, this.store)
      } catch (err) {
        console.error('Error executing ui-code:', err)
      }
    }

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
        widget,  // CRC: crc-Variable.md - widget reference
      })
      .then((id) => {
        childVarId = id
        const unbindValue = this.store.watch(id, (_v, value) => executeCode(value))

        // Initial execution from cached value
        const current = this.store.get(id)
        if (current?.value) executeCode(current.value)

        // Register with Widget
        widget.registerBinding('ui-code', id, () => {
          unbindValue()
          this.store.destroy(id)
        })
      })
      .catch((err) => {
        console.error('Failed to create code binding variable:', err)
      })
  }

  // Create an HTML binding (sets innerHTML or replaces element)
  // Spec: viewdefs.md - ui-html binding
  // CRC: crc-HtmlBinding.md
  private createHtmlBinding(
    varId: number,
    path: string,
    widget: Widget
  ): void {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    if (parsed.options.props?.['scrollOnOutput'] === 'true') {
      widget.scrollOnOutput = true
    }

    if (!properties['access']) {
      properties['access'] = 'r'
    }

    const replaceMode = parsed.options.props?.['replace'] === 'true'
    const originalElementId = widget.elementId
    let trackedElementIds: string[] = [originalElementId]

    // Parse HTML string into DOM nodes
    function parseHtml(html: string): Node[] {
      const container = document.createElement('div')
      container.innerHTML = html
      return Array.from(container.childNodes)
    }

    // Assign IDs to element nodes (first gets original ID, rest get vended IDs)
    function assignElementIds(elements: Element[]): string[] {
      return elements.map((el, index) => {
        el.id = index === 0 ? originalElementId : vendElementId()
        return el.id
      })
    }

    // Create a hidden placeholder span with original ID
    function createPlaceholder(): HTMLSpanElement {
      const placeholder = document.createElement('span')
      placeholder.id = originalElementId
      placeholder.style.display = 'none'
      return placeholder
    }

    // Wrap nodes in a span with original ID
    function wrapInSpan(nodes: Node[]): HTMLSpanElement {
      const wrapper = document.createElement('span')
      wrapper.id = originalElementId
      for (const node of nodes) {
        wrapper.appendChild(node)
      }
      return wrapper
    }

    // Insert nodes before a reference node (preserves order)
    function insertNodesBeforeRef(parent: Node, nodes: Node[], refNode: Node | null): void {
      for (const node of nodes) {
        parent.insertBefore(node, refNode)
      }
    }

    // Standard mode: set innerHTML
    const updateInnerHtml = (value: unknown): void => {
      const el = document.getElementById(originalElementId)
      if (!el) return
      el.innerHTML = (value ?? '').toString()
      this.addScrollNotification(varId)
    }

    // Replace mode: replace element(s) with new HTML
    const updateReplaceHtml = (value: unknown): void => {
      const html = (value ?? '').toString()

      // Find anchor element and save insertion point before removal
      const anchorEl = document.getElementById(trackedElementIds[0])
      if (!anchorEl?.parentNode) return
      const parent = anchorEl.parentNode
      const insertionRef = anchorEl.nextSibling

      // Remove all currently tracked elements
      for (const id of trackedElementIds) {
        document.getElementById(id)?.remove()
      }

      const nodes = parseHtml(html)
      const elementNodes = nodes.filter((n): n is Element => n instanceof Element)

      // Determine what to insert based on parsed content
      if (nodes.length === 0) {
        // Empty content: use hidden placeholder
        parent.insertBefore(createPlaceholder(), insertionRef)
        trackedElementIds = [originalElementId]
      } else if (elementNodes.length === 0) {
        // Text/comment nodes only: wrap in span
        parent.insertBefore(wrapInSpan(nodes), insertionRef)
        trackedElementIds = [originalElementId]
      } else {
        // Has element nodes: assign IDs and insert all nodes
        trackedElementIds = assignElementIds(elementNodes)
        insertNodesBeforeRef(parent, nodes, insertionRef)
      }

      this.addScrollNotification(varId)
    }

    const update = replaceMode ? updateReplaceHtml : updateInnerHtml

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
        widget,
      })
      .then((id) => {
        const unbindValue = this.store.watch(id, (_v, value) => update(value))

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) update(current.value)

        // Register with Widget
        widget.registerBinding('ui-html', id, () => {
          unbindValue()
          this.store.destroy(id)
          // For replace mode, remove all tracked elements
          if (replaceMode) {
            for (const elemId of trackedElementIds) {
              const el = document.getElementById(elemId)
              if (el) el.remove()
            }
          }
        })
      })
      .catch((err) => {
        console.error('Failed to create HTML binding variable:', err)
      })
  }

  // Normalize keypress attribute key name to browser event.key value
  // Spec: viewdefs.md - ui-event-keypress-* bindings
  // CRC: crc-EventBinding.md - Key name normalization
  private normalizeKeyName(attrKey: string): string {
    const keyMap: Record<string, string> = {
      enter: 'Enter',
      escape: 'Escape',
      left: 'ArrowLeft',
      right: 'ArrowRight',
      up: 'ArrowUp',
      down: 'ArrowDown',
      tab: 'Tab',
      space: ' ',
    }
    const lower = attrKey.toLowerCase()
    return keyMap[lower] ?? lower // Single letters remain lowercase
  }

  // Check if keyboard event matches target key (case-insensitive for letters)
  // CRC: crc-EventBinding.md - matchesTargetKey
  private matchesTargetKey(event: KeyboardEvent, targetKey: string): boolean {
    const normalizedTarget = this.normalizeKeyName(targetKey)
    // For single letters, do case-insensitive comparison
    if (normalizedTarget.length === 1) {
      return event.key.toLowerCase() === normalizedTarget.toLowerCase()
    }
    return event.key === normalizedTarget
  }

  // Known modifier keys for keypress bindings
  // Spec: viewdefs.md - ui-event-keypress-* modifiers
  // CRC: crc-BindingEngine.md - isModifierKey
  private static readonly MODIFIER_KEYS = new Set(['ctrl', 'shift', 'alt', 'meta'])

  // Parse keypress attribute suffix into modifiers and key
  // e.g., "ctrl-shift-enter" → { modifiers: Set(['ctrl', 'shift']), key: 'enter' }
  // CRC: crc-BindingEngine.md - parseKeypressAttribute
  private parseKeypressAttribute(suffix: string): { modifiers: Set<string>; key: string } {
    const parts = suffix.toLowerCase().split('-')
    const modifiers = new Set<string>()
    let key = ''

    for (const part of parts) {
      if (BindingEngine.MODIFIER_KEYS.has(part)) {
        modifiers.add(part)
      } else {
        // Last non-modifier part is the key
        key = part
      }
    }

    return { modifiers, key }
  }

  // Check if keyboard event modifiers match exactly (all required, no extras)
  // Spec: viewdefs.md - Modifier matching is exact
  // CRC: crc-EventBinding.md - matchesModifiers
  private matchesModifiers(event: KeyboardEvent, requiredModifiers: Set<string>): boolean {
    const eventModifiers = new Set<string>()
    if (event.ctrlKey) eventModifiers.add('ctrl')
    if (event.shiftKey) eventModifiers.add('shift')
    if (event.altKey) eventModifiers.add('alt')
    if (event.metaKey) eventModifiers.add('meta')

    // Check exact match: same size and all required modifiers present
    if (eventModifiers.size !== requiredModifiers.size) return false
    for (const mod of requiredModifiers) {
      if (!eventModifiers.has(mod)) return false
    }
    return true
  }

  // Sync ui-value before sending event if element value differs from cached
  // Spec: viewdefs.md - Event Bindings (value sync with ui-value)
  // CRC: crc-EventBinding.md - Event Update Behavior
  private syncValueBeforeEvent(element: Element, widget: Widget): void {
    const valueVarId = widget.getVariableId('ui-value')
    if (valueVarId === undefined) return

    // Get element's current value
    const elementValue = (element as HTMLInputElement).value
    if (elementValue === undefined) return

    // Get cached variable value
    const variable = this.store.get(valueVarId)
    if (!variable) return

    // Compare and sync if different
    if (variable.value !== elementValue) {
      this.store.update(valueVarId, elementValue)
    }
  }

  // Create an event binding (custom events like ui-event="action?eventName")
  // Creates an action variable and invokes it when the specified event fires
  // Spec: viewdefs.md - Event Bindings, ui-event-keypress-*
  private createEventBinding(
    element: Element,
    varId: number,
    actionExpr: string,
    eventName: string,
    widget: Widget
  ): void {
    // Handle keypress-specific bindings (ui-event-keypress-enter, etc.)
    // CRC: crc-EventBinding.md - Keypress Binding
    if (eventName.startsWith('keypress-')) {
      this.createKeypressBinding(element, varId, actionExpr, eventName, widget)
      return
    }

    // Parse action expression to build path (same as createActionBinding)
    const match = actionExpr.match(/^([\w.]+)\((.*)\)$/)
    if (!match) {
      console.error('Invalid action expression:', actionExpr)
      return
    }

    const [, methodPath, argsStr] = match
    const hasArgPlaceholder = argsStr === '_'
    const path = hasArgPlaceholder ? `${methodPath}(_)` : `${methodPath}()`

    const properties: Record<string, string> = {
      path,
      access: 'action',
    }

    // Spec: viewdefs.md - Event Bindings (value sync with ui-value)
    // Before sending event, sync ui-value if element value differs from cached
    const handler = (_event: Event) => {
      if (actionVarId !== null) {
        // Check for ui-value binding and sync if needed
        this.syncValueBeforeEvent(element, widget)
        // Then send the event
        this.store.update(actionVarId, null)
      }
    }

    element.addEventListener(eventName, handler)

    let actionVarId: number | null = null

    this.store
      .create({
        parentId: varId,
        properties,
        widget,  // CRC: crc-Variable.md - widget reference
      })
      .then((id) => {
        actionVarId = id
        // Register with Widget
        widget.registerBinding(`ui-event-${eventName}`, id, () => {
          element.removeEventListener(eventName, handler)
          this.store.destroy(id)
        })
      })
      .catch((err) => {
        console.error('Failed to create action variable:', err)
      })
  }

  // Create a keypress-specific binding (ui-event-keypress-enter, ui-event-keypress-ctrl-enter, etc.)
  // Listens on keydown, filters by target key and modifiers, updates variable with key name
  // Spec: viewdefs.md - ui-event-keypress-* bindings with modifiers
  // CRC: crc-EventBinding.md - Keypress Binding
  private createKeypressBinding(
    element: Element,
    varId: number,
    path: string,
    eventName: string,
    widget: Widget
  ): void {
    // Extract modifiers and key from eventName (e.g., "ctrl-enter" from "keypress-ctrl-enter")
    const suffix = eventName.substring(9) // Remove "keypress-"
    if (!suffix) {
      console.error('Invalid keypress binding - missing key name:', eventName)
      return
    }

    const { modifiers, key: targetKey } = this.parseKeypressAttribute(suffix)
    if (!targetKey) {
      console.error('Invalid keypress binding - no key found:', eventName)
      return
    }

    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    const pathWithoutOptions = parsed.segments.join('.')
    properties['path'] = pathWithoutOptions

    // Detect action type: no-arg action(), 1-arg action(_), or non-action
    // Check pathWithoutOptions, not raw path (which may have query params)
    const isNoArgAction = pathWithoutOptions.match(/\(\)$/)
    const isOneArgAction = pathWithoutOptions.match(/\(_\)$/)
    if (isNoArgAction || isOneArgAction) {
      properties['access'] = 'action'
    }

    let childVarId: number | null = null

    // Listen on keydown and filter by target key and modifiers
    // Spec: viewdefs.md - Event Bindings (value sync with ui-value), Modifier matching is exact
    const handler = (event: Event) => {
      const keyEvent = event as KeyboardEvent
      if (this.matchesTargetKey(keyEvent, targetKey) && this.matchesModifiers(keyEvent, modifiers)) {
        if (childVarId !== null) {
          // Sync ui-value before sending event
          this.syncValueBeforeEvent(element, widget)
          // No-arg actions get null; 1-arg actions and non-actions get key name
          this.store.update(childVarId, isNoArgAction ? null : targetKey.toLowerCase())
        }
      }
    }

    element.addEventListener('keydown', handler)

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
        widget,  // CRC: crc-Variable.md - widget reference
      })
      .then((id) => {
        childVarId = id
        // Register with Widget
        widget.registerBinding(`ui-event-${eventName}`, id, () => {
          element.removeEventListener('keydown', handler)
          this.store.destroy(id)
        })
      })
      .catch((err) => {
        console.error('Failed to create keypress binding variable:', err)
      })
  }

  // Create an action binding (click)
  // Creates an action variable for the method path and invokes it on click
  // Spec: resolver.md - Variable Access Property, Path Semantics
  private createActionBinding(
    element: Element,
    varId: number,
    actionExpr: string,
    widget: Widget
  ): void {
    // Parse action expression: methodName() or path.to.method()
    // The () is required for actions and indicates a method call
    const match = actionExpr.match(/^([\w.]+)\((.*)\)$/)
    if (!match) {
      console.error('Invalid action expression:', actionExpr)
      return
    }

    const [, methodPath, argsStr] = match

    // Build the path for the action variable
    // For methods without args: path() (calls method for side effects)
    // For methods with args placeholder: path(_) (calls method with value)
    const hasArgPlaceholder = argsStr === '_'
    const path = hasArgPlaceholder ? `${methodPath}(_)` : `${methodPath}()`

    // Properties for the action variable
    // Use "action" access: initial value not computed (avoids premature method invocation)
    const properties: Record<string, string> = {
      path,
      access: 'action',
    }

    let actionVarId: number | null = null

    const handler = (event: Event) => {
      event.preventDefault()
      if (actionVarId !== null) {
        // Invoke the action by updating the action variable
        // For () paths: the method is called for side effects (value ignored)
        // For (_) paths: the value is passed to the method
        this.store.update(actionVarId, null)
      }
    }

    element.addEventListener('click', handler)

    // Create the action variable asynchronously
    this.store
      .create({
        parentId: varId,
        properties,
        widget,  // CRC: crc-Variable.md - widget reference
      })
      .then((id) => {
        actionVarId = id
        // Register with Widget
        widget.registerBinding('ui-action', id, () => {
          element.removeEventListener('click', handler)
          this.store.destroy(id)
        })
      })
      .catch((err) => {
        console.error('Failed to create action variable:', err)
      })
  }
}
