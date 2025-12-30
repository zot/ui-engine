// Binding engine for ui-* attributes
// CRC: crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md
// Spec: viewdefs.md

import { VariableStore, VariableError } from './connection'

export interface Binding {
  element: Element
  unbind: () => void
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

function isSlInput(element: any) {
  return (
    element instanceof HTMLElement &&
    (element.nodeName == 'SL-INPUT' || element.nodeName == 'SL-TEXTAREA')
  )
}

// Parse a path like "father.name?create=Person&wrapper=ViewList&item=ContactPresenter"
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
  private bindings: Map<Element, Binding[]> = new Map()

  constructor(store: VariableStore) {
    this.store = store
  }

  // Bind all ui-* attributes on an element
  bindElement(element: Element, contextVarId: number): void {
    const elementBindings: Binding[] = []

    // ui-value binding
    const uiValue = element.getAttribute('ui-value')
    if (uiValue) {
      const binding = this.createValueBinding(element, contextVarId, uiValue)
      if (binding) elementBindings.push(binding)
    }

    // ui-attr-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-attr-')) {
        const targetAttr = attr.name.substring(8) // Remove "ui-attr-"
        const binding = this.createAttrBinding(
          element,
          contextVarId,
          attr.value,
          targetAttr
        )
        if (binding) elementBindings.push(binding)
      }
    }

    // ui-class-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-class-')) {
        const className = attr.name.substring(9) // Remove "ui-class-"
        const binding = this.createClassBinding(
          element,
          contextVarId,
          attr.value,
          className
        )
        if (binding) elementBindings.push(binding)
      }
    }

    // ui-style-*-* bindings (e.g., ui-style-background-color)
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-style-')) {
        const styleProp = attr.name.substring(9) // Remove "ui-style-"
        const binding = this.createStyleBinding(
          element,
          contextVarId,
          attr.value,
          styleProp
        )
        if (binding) elementBindings.push(binding)
      }
    }

    // ui-event-* bindings
    for (const attr of Array.from(element.attributes)) {
      if (attr.name.startsWith('ui-event-')) {
        const eventName = attr.name.substring(9) // Remove "ui-event-"
        const binding = this.createEventBinding(
          element,
          contextVarId,
          attr.value,
          eventName
        )
        if (binding) elementBindings.push(binding)
      }
    }

    // ui-action binding (shorthand for click action)
    const uiAction = element.getAttribute('ui-action')
    if (uiAction) {
      const binding = this.createActionBinding(element, contextVarId, uiAction)
      if (binding) elementBindings.push(binding)
    }

    if (elementBindings.length > 0) {
      this.bindings.set(element, elementBindings)
    }

    // Recursively bind children
    for (const child of Array.from(element.children)) {
      this.bindElement(child, contextVarId)
    }
  }

  // Unbind all bindings from an element and its children
  unbindElement(element: Element): void {
    const elementBindings = this.bindings.get(element)
    if (elementBindings) {
      elementBindings.forEach((b) => b.unbind())
      this.bindings.delete(element)
    }

    for (const child of Array.from(element.children)) {
      this.unbindElement(child)
    }
  }

  // Create a value binding (sets textContent or value, and handles changes)
  // Spec: viewdefs.md - Nullish path handling with error indicators
  // Spec: libraries.md - Input update behavior (blur by default, keypress for immediate)
  // ARCHITECTURE.md: Frontend creates child variables for paths
  private createValueBinding(
    element: Element,
    varId: number,
    path: string
  ): Binding | null {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    // Check if keypress mode is enabled (send updates on every keypress vs blur)
    const useKeypress = parsed.options.props?.['keypress'] === 'true'

    // Create a child variable for this path
    // The server will resolve the path and send back the value
    let childVarId: number | null = null
    let unbindValue: (() => void) | null = null
    let unbindError: (() => void) | null = null
    // Custom elements (tagNames with hyphens) may not be upgraded yet when binding runs,
    // so 'value' in element would return false. Assume custom elements have value property.
    // Exclude buttons - they have a value property for forms but we want textContent.
    const isCustomElement = element.tagName.includes('-')
    const isButton = element instanceof HTMLButtonElement
    const editableValue =
      !isButton &&
      (element instanceof HTMLInputElement ||
        element instanceof HTMLTextAreaElement ||
        element instanceof HTMLSelectElement ||
        isSlInput(element) ||
        isCustomElement ||
        'value' in element)
    const update = editableValue
      ? (value: unknown) => {
          // Preserve number type for components like sl-rating, sl-range
          if (typeof value === 'number') {
            (element as any).value = value
          } else {
            (element as any).value = value?.toString() ?? ''
          }
        }
      : (value: unknown) => (element.textContent = value?.toString() ?? '')

    // Handle error state changes - add/remove ui-error class and data-ui-error-* attributes
    const updateError = (error: VariableError | null) => {
      if (error) {
        element.classList.add('ui-error')
        element.setAttribute('data-ui-error-code', error.code)
        element.setAttribute('data-ui-error-description', error.description)
      } else {
        element.classList.remove('ui-error')
        element.removeAttribute('data-ui-error-code')
        element.removeAttribute('data-ui-error-description')
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

    if (!editableValue && properties['access'] === undefined) {
      properties.access = 'r'
    }

    // Create the child variable asynchronously
    this.store
      .create({
        parentId: varId,
        properties,
      })
      .then((id) => {
        childVarId = id
        // Watch the child variable for value updates
        unbindValue = this.store.watch(id, (_v, value) => update(value))
        unbindError = this.store.watchErrors(id, updateError)

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) update(current.value)
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
      tagLower === 'sl-input' || tagLower === 'sl-textarea'

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

    return {
      element,
      unbind: () => {
        if (unbindValue) unbindValue()
        if (unbindError) unbindError()
        // Remove native input listener
        if (nativeEventType) {
          element.removeEventListener(nativeEventType, inputHandler)
        }
        // Remove Shoelace listener
        if (shoelaceEventType) {
          element.removeEventListener(shoelaceEventType, shoelaceHandler)
        }
        // Remove custom widget listener
        element.removeEventListener('ui-value-change', changeHandler)
        // Destroy the child variable
        if (childVarId !== null) {
          this.store.destroy(childVarId)
        }
        // Clean up error state on unbind
        element.classList.remove('ui-error')
        element.removeAttribute('data-ui-error-code')
        element.removeAttribute('data-ui-error-description')
      },
    }
  }

  // Create an attribute binding
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private createAttrBinding(
    element: Element,
    varId: number,
    path: string,
    targetAttr: string
  ): Binding | null {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    let childVarId: number | null = null
    let unbindValue: (() => void) | null = null

    const update = (value: unknown) => {
      if (value !== null && value !== undefined && value !== false) {
        element.setAttribute(targetAttr, value.toString())
      } else {
        element.removeAttribute(targetAttr)
      }
    }

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
      })
      .then((id) => {
        childVarId = id
        unbindValue = this.store.watch(id, (_v, value) => update(value))

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) update(current.value)
      })
      .catch((err) => {
        console.error('Failed to create attr binding variable:', err)
      })

    return {
      element,
      unbind: () => {
        if (unbindValue) unbindValue()
        if (childVarId !== null) {
          this.store.destroy(childVarId)
        }
      },
    }
  }

  // Create a class binding
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private createClassBinding(
    element: Element,
    varId: number,
    path: string,
    className: string
  ): Binding | null {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')

    let childVarId: number | null = null
    let unbindValue: (() => void) | null = null

    const update = (value: unknown) => {
      if (value) {
        element.classList.add(className)
      } else {
        element.classList.remove(className)
      }
    }

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
      })
      .then((id) => {
        childVarId = id
        unbindValue = this.store.watch(id, (_v, value) => update(value))

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) update(current.value)
      })
      .catch((err) => {
        console.error('Failed to create class binding variable:', err)
      })

    return {
      element,
      unbind: () => {
        if (unbindValue) unbindValue()
        if (childVarId !== null) {
          this.store.destroy(childVarId)
        }
      },
    }
  }

  // Create a style binding
  // Spec: viewdefs.md - Path Resolution: Server-Side Only
  private createStyleBinding(
    element: Element,
    varId: number,
    path: string,
    styleProp: string
  ): Binding | null {
    const parsed = parsePath(path)
    const properties = pathOptionsToProperties(parsed.options)
    properties['path'] = parsed.segments.join('.')
    const htmlElement = element as HTMLElement

    let childVarId: number | null = null
    let unbindValue: (() => void) | null = null

    const update = (value: unknown) => {
      if (value !== null && value !== undefined) {
        htmlElement.style.setProperty(styleProp, value.toString())
      } else {
        htmlElement.style.removeProperty(styleProp)
      }
    }

    // Create a child variable for this path
    this.store
      .create({
        parentId: varId,
        properties,
      })
      .then((id) => {
        childVarId = id
        unbindValue = this.store.watch(id, (_v, value) => update(value))

        // Initial update from cached value
        const current = this.store.get(id)
        if (current) update(current.value)
      })
      .catch((err) => {
        console.error('Failed to create style binding variable:', err)
      })

    return {
      element,
      unbind: () => {
        if (unbindValue) unbindValue()
        if (childVarId !== null) {
          this.store.destroy(childVarId)
        }
      },
    }
  }

  // Create an event binding (custom events like ui-event="action?eventName")
  // Creates an action variable and invokes it when the specified event fires
  private createEventBinding(
    element: Element,
    varId: number,
    actionExpr: string,
    eventName: string
  ): Binding | null {
    // Parse action expression to build path (same as createActionBinding)
    const match = actionExpr.match(/^([\w.]+)\((.*)\)$/)
    if (!match) {
      console.error('Invalid action expression:', actionExpr)
      return null
    }

    const [, methodPath, argsStr] = match
    const hasArgPlaceholder = argsStr === '_'
    const path = hasArgPlaceholder ? `${methodPath}(_)` : `${methodPath}()`

    const properties: Record<string, string> = {
      path,
      access: 'action',
    }

    let actionVarId: number | null = null

    this.store
      .create({
        parentId: varId,
        properties,
      })
      .then((id) => {
        actionVarId = id
      })
      .catch((err) => {
        console.error('Failed to create action variable:', err)
      })

    const handler = (_event: Event) => {
      if (actionVarId !== null) {
        this.store.update(actionVarId, null)
      }
    }

    element.addEventListener(eventName, handler)

    return {
      element,
      unbind: () => {
        element.removeEventListener(eventName, handler)
        if (actionVarId !== null) {
          this.store.destroy(actionVarId)
        }
      },
    }
  }

  // Create an action binding (click)
  // Creates an action variable for the method path and invokes it on click
  // Spec: resolver.md - Variable Access Property, Path Semantics
  private createActionBinding(
    element: Element,
    varId: number,
    actionExpr: string
  ): Binding | null {
    // Parse action expression: methodName() or path.to.method()
    // The () is required for actions and indicates a method call
    const match = actionExpr.match(/^([\w.]+)\((.*)\)$/)
    if (!match) {
      console.error('Invalid action expression:', actionExpr)
      return null
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

    // Create the action variable asynchronously
    this.store
      .create({
        parentId: varId,
        properties,
      })
      .then((id) => {
        actionVarId = id
      })
      .catch((err) => {
        console.error('Failed to create action variable:', err)
      })

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

    return {
      element,
      unbind: () => {
        element.removeEventListener('click', handler)
        // Destroy the action variable
        if (actionVarId !== null) {
          this.store.destroy(actionVarId)
        }
      },
    }
  }
}
