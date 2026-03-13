import { Auth } from "./auth";

const BASE = "/api";

/**
 * Discriminated union for API call results.
 * Check `result.ok` before accessing `result.data` — TypeScript narrows automatically.
 */
export type Result<T> = { ok: true; data: T } | { ok: false; error: string };

/** Structured SSE event from the classification workflow. */
export interface ExecutionEvent {
  type: string;
  timestamp: string;
  data: Record<string, unknown>;
}

async function authHeaders(): Promise<Record<string, string>> {
  if (!Auth.isEnabled()) return {};
  const token = await Auth.getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

/**
 * Generic fetch wrapper. Prepends `/api` to the path.
 *
 * Non-2xx responses return the body as the error message.
 * 204 responses return `undefined` as data.
 * Override `parse` for non-JSON responses (default: `res.json()`).
 */
export async function request<T>(
  path: string,
  init?: RequestInit,
  parse: (res: Response) => Promise<T> = (res) => res.json(),
): Promise<Result<T>> {
  try {
    const headers = { ...init?.headers, ...(await authHeaders()) };
    const opts: RequestInit = { ...init, headers };
    let res = await fetch(`${BASE}${path}`, opts);

    if (res.status === 401 && Auth.isEnabled()) {
      const token = await Auth.getToken(true);
      if (token) {
        opts.headers = { ...opts.headers, Authorization: `Bearer ${token}` };
        res = await fetch(`${BASE}${path}`, opts);
      }
      if (res.status === 401) {
        await Auth.login();
        return { ok: false, error: "Authentication required" };
      }
    }

    if (!res.ok) {
      const text = await res.text();
      return { ok: false, error: text || res.statusText };
    }
    if (res.status === 204) {
      return { ok: true, data: undefined as T };
    }
    return { ok: true, data: await parse(res) };
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
}

/**
 * Callbacks for SSE stream consumption.
 *
 * `onEvent` receives the event type (from `event:` lines, default `'message'`)
 * paired with the raw `data:` line content.
 */
export interface StreamOptions {
  onEvent: (type: string, data: string) => void;
  onError?: (error: string) => void;
  onComplete?: () => void;
  signal?: AbortSignal;
}

export function stream(
  path: string,
  options: StreamOptions,
  init?: RequestInit,
): AbortController {
  const controller = new AbortController();
  const signal = options.signal ?? controller.signal;

  (async () => {
    try {
      const headers = { ...init?.headers, ...(await authHeaders()) };
      let opts: RequestInit = { ...init, headers, signal };
      let res = await fetch(`${BASE}${path}`, opts);

      if (res.status === 401 && Auth.isEnabled()) {
        const token = await Auth.getToken(true);
        if (token) {
          opts = {
            ...opts,
            headers: { ...opts.headers, Authorization: `Bearer ${token}` },
          };
          res = await fetch(`${BASE}${path}`, opts);
        }
        if (res.status === 401) {
          await Auth.login();
          return;
        }
      }

      if (!res.ok) {
        const text = await res.text();
        options.onError?.(text || res.statusText);
        return;
      }

      const reader = res.body?.getReader();
      if (!reader) {
        options.onError?.("No response body");
        return;
      }

      const decoder = new TextDecoder();
      let buffer = "";
      let currentEvent = "message";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() ?? "";

        for (const line of lines) {
          if (line.startsWith("event: ")) {
            currentEvent = line.slice(7).trim();
          } else if (line.startsWith("data: ")) {
            const data = line.slice(6).trim();
            options.onEvent(currentEvent, data);
            currentEvent = "message";
          }
        }
      }

      options.onComplete?.();
    } catch (err: unknown) {
      if ((err as Error).name !== "AbortError") {
        options.onError?.((err as Error).message);
      }
    }
  })();

  return controller;
}

/** Paginated response from the server. */
export interface PageResult<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

/** Converts pagination params to a query string (e.g., `?page=1&page_size=20`). */
export function toQueryString<T extends object>(params: T): string {
  const entries = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== "")
    .map(
      ([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`,
    );

  return entries.length > 0 ? `?${entries.join("&")}` : "";
}
