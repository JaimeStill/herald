import type { RouteConfig, RouteMatch } from "./types";

let routerInstance: Router | null = null;

/** Programmatic navigation from anywhere in the app. */
export function navigate(path: string): void {
  routerInstance?.navigate(path);
}

/**
 * History API router. Matches URL paths against the route table,
 * mounts the corresponding component into a container element,
 * and sets path/query params as HTML attributes on the mounted component.
 */
export class Router {
  private container: HTMLElement;
  private basePath: string;
  private routes: Record<string, RouteConfig>;

  constructor(containerId: string, routes: Record<string, RouteConfig>) {
    const el = document.getElementById(containerId);
    if (!el) throw new Error(`Container #${containerId} not found`);

    this.routes = routes;

    this.container = el;
    this.basePath =
      document
        .querySelector("base")
        ?.getAttribute("href")
        ?.replace(/\/$/, "") ?? "/app";

    routerInstance = this;
  }

  /** Navigate to a path, optionally pushing to browser history. */
  navigate(path: string, pushState: boolean = true): void {
    const [pathPart, queryPart] = path.split("?");
    const normalized = this.normalizePath(pathPart);
    const query = this.parseQuery(queryPart);
    const match = this.match(normalized, query);

    if (pushState) {
      let fullPath = `${this.basePath}/${normalized}`.replace(/\/+/g, "/");
      if (queryPart) fullPath += `?${queryPart}`;
      history.pushState(null, "", fullPath);
    }

    document.title = `${match.config.title} - Herald`;
    this.mount(match);
  }

  /** Initialize the router: mount the current URL and listen for popstate. */
  start(): void {
    this.navigate(this.currentPath(), false);

    window.addEventListener("popstate", () => {
      this.navigate(this.currentPath(), false);
    });
  }

  private currentPath(): string {
    const pathname = location.pathname;

    if (pathname.startsWith(this.basePath))
      return pathname.slice(this.basePath.length).replace(/^\//, "");

    return pathname.replace(/^\//, "");
  }

  private match(path: string, query: Record<string, string>): RouteMatch {
    const segments = path.split("/").filter(Boolean);

    if (this.routes[path]) return { config: this.routes[path], params: {}, query };

    for (const [pattern, config] of Object.entries(this.routes)) {
      if (pattern === "*") continue;

      const patternSegments = pattern.split("/").filter(Boolean);

      if (patternSegments.length !== segments.length) continue;

      const params: Record<string, string> = {};
      let matched = true;

      for (let i = 0; i < patternSegments.length; i++) {
        const pat = patternSegments[i];
        const seg = segments[i];

        if (pat.startsWith(":")) {
          params[pat.slice(1)] = seg;
        } else if (pat !== seg) {
          matched = false;
          break;
        }
      }

      if (matched) {
        return { config, params, query };
      }
    }

    return { config: this.routes["*"], params: { path }, query };
  }

  private mount(match: RouteMatch): void {
    const update = () => {
      this.container.innerHTML = "";
      const el = document.createElement(match.config.component);

      for (const [key, value] of Object.entries(match.params)) {
        el.setAttribute(key, value);
      }

      for (const [key, value] of Object.entries(match.query)) {
        el.setAttribute(key, value);
      }

      this.container.appendChild(el);
    };

    if (document.startViewTransition) {
      document.startViewTransition(update);
    } else {
      update();
    }
  }

  private normalizePath(path: string): string {
    let normalized = path.replace(/^\//, "");
    const baseWithoutSlash = this.basePath.replace(/^\//, "");

    if (normalized.startsWith(baseWithoutSlash))
      normalized = normalized.slice(baseWithoutSlash.length).replace(/^\//, "");

    return normalized;
  }

  private parseQuery(queryString?: string): Record<string, string> {
    if (!queryString) return {};

    const params = new URLSearchParams(queryString);
    const result: Record<string, string> = {};
    for (const [key, value] of params) {
      result[key] = value;
    }
    return result;
  }
}
