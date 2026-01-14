// URL routing for session navigation
// CRC: crc-Router.md
// Spec: interfaces.md

/** Route mapping path to presenter variable */
export interface Route {
  path: string;
  variableId: number;
}

/** Router handles URL routing for a session */
export class Router {
  private sessionId: string;
  private routes: Map<string, Route> = new Map();

  constructor(sessionId: string) {
    this.sessionId = sessionId;
  }

  /** Register a URL path with a presenter variable */
  register(path: string, variableId: number): void {
    path = normalizePath(path);
    this.routes.set(path, { path, variableId });
  }

  /** Unregister a URL path mapping */
  unregister(path: string): boolean {
    path = normalizePath(path);
    return this.routes.delete(path);
  }

  /** Find presenter variable ID for a URL path */
  resolve(path: string): number | null {
    path = normalizePath(path);
    const route = this.routes.get(path);
    return route ? route.variableId : null;
  }

  /** Check if URL matches any registered pattern */
  match(path: string): boolean {
    return this.resolve(path) !== null;
  }

  /** Build full URL for a presenter path */
  buildUrl(path: string): string {
    path = normalizePath(path);
    if (path === '/') {
      return '/' + this.sessionId;
    }
    return '/' + this.sessionId + path;
  }

  /** Check if path was explicitly registered */
  isRegisteredPath(path: string): boolean {
    path = normalizePath(path);
    return this.routes.has(path);
  }

  /** Get session ID */
  getSessionId(): string {
    return this.sessionId;
  }

  /** Get all registered routes */
  getRoutes(): Route[] {
    return Array.from(this.routes.values());
  }
}

/**
 * Parse URL to extract session ID and path.
 * Input: /SESSION-ID/some/path
 * Returns: { sessionId: "SESSION-ID", path: "/some/path" }
 */
export function parseUrl(urlPath: string): { sessionId: string; path: string } {
  // Remove leading slash
  urlPath = urlPath.replace(/^\//, '');

  // Split on first slash
  const slashIndex = urlPath.indexOf('/');
  if (slashIndex === -1) {
    return { sessionId: urlPath, path: '/' };
  }

  return {
    sessionId: urlPath.substring(0, slashIndex),
    path: '/' + urlPath.substring(slashIndex + 1),
  };
}

/**
 * Get session ID from cookie.
 * Returns empty string if cookie not found.
 */
function getSessionIdFromCookie(): string {
  const match = document.cookie.match(/(?:^|; )ui-session=([^;]*)/);
  return match ? match[1] : '';
}

/**
 * Extract session ID from cookie or URL.
 * Cookie takes precedence (set by server).
 * Throws if no session ID found.
 */
export function getSessionIdFromLocation(): string {
  // Cookie takes precedence (always set by server)
  const cookieSessionId = getSessionIdFromCookie();
  if (cookieSessionId) {
    return cookieSessionId;
  }
  // Fall back to URL path
  const { sessionId } = parseUrl(window.location.pathname);
  if (!sessionId) {
    throw new Error('No session ID in URL or cookie');
  }
  return sessionId;
}

/**
 * Navigate to a path within the current session using History API.
 * @param router Router instance
 * @param path Path to navigate to (without session ID)
 * @param replace If true, replaces current history entry
 */
export function navigateTo(router: Router, path: string, replace = false): void {
  const url = router.buildUrl(path);
  if (replace) {
    window.history.replaceState(null, '', url);
  } else {
    window.history.pushState(null, '', url);
  }
  window.dispatchEvent(new PopStateEvent('popstate'));
}

/** Normalize path to have leading slash and no trailing slash */
function normalizePath(path: string): string {
  if (!path) {
    return '/';
  }
  if (!path.startsWith('/')) {
    path = '/' + path;
  }
  if (path !== '/' && path.endsWith('/')) {
    path = path.slice(0, -1);
  }
  return path;
}
