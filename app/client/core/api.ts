const BASE = '/api';

export type Result<T> =
  | { ok: true; data: T }
  | { ok: false; error: string };

export async function request<T>(
  path: string,
  init?: RequestInit,
  parse: (res: Response) => Promise<T> = (res) => res.json()
): Promise<Result<T>> {
  try {
    const res = await fetch(`${BASE}${path}`, init);
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

export interface StreamOptions {
  onMessage: (data: string) => void;
  onError?: (error: string) => void;
  onComplete?: () => void;
  signal?: AbortSignal;
}

export function stream(
  path: string,
  options: StreamOptions
): AbortController {
  const controller = new AbortController();
  const signal = options.signal ?? controller.signal;

  fetch(`${BASE}${path}`, { signal })
    .then(async (res) => {
      if (!res.ok) {
        const text = await res.text();
        options.onError?.(text || res.statusText);
        return;
      }

      const reader = res.body?.getReader();
      if (!reader) {
        options.onError?.('No response body');
        return;
      }

      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6).trim();
            if (data === '[DONE]') {
              options.onComplete?.();
              return;
            }
            options.onMessage(data);
          }
        }
      }

      options.onComplete?.();
    })
    .catch((err: Error) => {
      if (err.name !== 'AbortError') {
        options.onError?.(err.message);
      }
    });

  return controller;
}

export interface PageResult<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface PageRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
}

export function toQueryString(params: PageRequest): string {
  const entries = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== '')
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);

  return entries.length > 0
    ? `?${entries.join('&')}`
    : '';
}
