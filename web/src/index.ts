// Main entry point - exports all public API
// Spec: libraries.md

export { UIApp, initUIApp } from './app';
export { Connection, VariableStore } from './connection';
export { BindingEngine, parsePath, resolvePath } from './binding';
export type { Binding, ParsedPath, PathOptions } from './binding';
export type {
  Message,
  MessageType,
  CreateMessage,
  DestroyMessage,
  UpdateMessage,
  WatchMessage,
  ErrorMessage,
  Response,
  VariableData,
  ObjectData,
} from './protocol';
export { createMessage, parseMessage, encodeMessage } from './protocol';
export type { Variable, ObjectReference } from './variable';
export { isObjectReference, getObjectReferenceId } from './variable';
export type { Viewdef } from './viewdef';
export {
  parseKey,
  buildKey,
  parseViewdef,
  createViewdef,
  cloneViewdefContent,
  getViewdefKey,
} from './viewdef';
export { ViewdefStore } from './viewdef_store';
export type { PendingView } from './viewdef_store';
export { View, createView } from './view';
export { ViewList, createViewList } from './viewlist';
export type { ViewListDelegate } from './viewlist';
export { ViewRenderer, createViewRenderer } from './renderer';
export { MessageBatcher, Priority, parsePrioritySuffix, parseBatch } from './batcher';
export { Router, parseUrl, getSessionIdFromLocation, navigateTo } from './router';
export type { Route } from './router';
export {
  parsePath as parseVariablePath,
  pathToString,
  resolve as resolvePath2,
  resolveForWrite,
  hasStandardVariable,
  getStandardVariable,
  SegmentType,
} from './path';
export type { Segment, ParsedPath as VariablePath, ResolveResult, WriteTarget } from './path';
export {
  registerWidget,
  getWidgetHandler,
  hasWidgetHandler,
  bindWidget,
  updateWidget,
} from './widgets';
export type { WidgetHandler } from './widgets';
