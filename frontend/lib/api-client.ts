// Browser-side API calls go through the same-origin /api rewrite, so the
// session cookie is sent automatically.

export type Result<T = unknown> = { data?: T; error?: string; status: number };

// API messages are lowercase fragments ("only 2 seats left in that time range");
// show them as sentences.
function sentence(message: string): string {
  const s = message.trim();
  if (!s) return "Something went wrong. Try again.";
  const capped = s[0].toUpperCase() + s.slice(1);
  return /[.!?]$/.test(capped) ? capped : `${capped}.`;
}

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
    if (res.status === 401) return { status: 401, error: "Your session has ended. Log in again." };
    return { status: res.status, error: sentence(json?.error?.message ?? "") };
  }
  return { status: res.status, data: json as T };
}
