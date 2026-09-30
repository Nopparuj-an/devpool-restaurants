// All API calls come from the browser and go through the same-origin /api
// path (a Next rewrite to Go), so the HttpOnly session cookie is sent
// automatically and never touches JavaScript. See ADR-0013.

export type Result<T = unknown> = { data?: T; error?: string; status: number };

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

// API messages are lowercase fragments ("only 2 seats left in that time range");
// show them as sentences.
function sentence(message: string): string {
  const s = message.trim();
  if (!s) return "Something went wrong. Try again.";
  const capped = s[0].toUpperCase() + s.slice(1);
  return /[.!?]$/.test(capped) ? capped : `${capped}.`;
}

// Reads for TanStack Query: resolve with the body or throw an ApiError.
export async function get<T>(path: string): Promise<T> {
  const res = await fetch(`/api${path}`, { credentials: "same-origin" });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error?.code ?? "unknown", sentence(body?.error?.message ?? res.statusText));
  }
  return res.json();
}

// Like get, but 404 becomes null.
export async function getOrNull<T>(path: string): Promise<T | null> {
  try {
    return await get<T>(path);
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) return null;
    throw e;
  }
}

// Writes: never throw, so forms can show the error inline.
export async function api<T = unknown>(method: string, path: string, body?: unknown): Promise<Result<T>> {
  const init: RequestInit = { method, credentials: "same-origin" };
  if (body instanceof FormData) {
    init.body = body;
  } else if (body !== undefined) {
    init.body = JSON.stringify(body);
    init.headers = { "Content-Type": "application/json" };
  }
  let res: Response;
  try {
    res = await fetch(`/api${path}`, init);
  } catch {
    return { status: 0, error: "Can't reach the server. Check your connection and try again." };
  }
  if (res.status === 204) return { status: 204 };
  const json = await res.json().catch(() => null);
  if (!res.ok) {
    const code = json?.error?.code;
    if (code === "session_expired") return { status: 401, error: "Your session has ended. Log in again." };
    if (code === "unauthorized") return { status: 401, error: "You need to log in first." };
    return { status: res.status, error: sentence(json?.error?.message ?? "") };
  }
  return { status: res.status, data: json as T };
}
