import "server-only";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import type { Account } from "./types";

// Server components call the Go API directly (not through the /api rewrite)
// and forward the browser's session cookie.
const API_URL = process.env.API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

export async function apiGet<T>(path: string): Promise<T> {
  const cookie = (await cookies()).toString();
  const res = await fetch(`${API_URL}/api${path}`, {
    headers: cookie ? { cookie } : {},
    cache: "no-store",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(res.status, body?.error?.code ?? "unknown", body?.error?.message ?? res.statusText);
  }
  return res.json();
}

// Like apiGet, but 404 becomes null.
export async function apiGetOrNull<T>(path: string): Promise<T | null> {
  try {
    return await apiGet<T>(path);
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) return null;
    throw e;
  }
}

export async function getAccount(): Promise<Account | null> {
  if (!(await cookies()).has("session")) return null;
  try {
    return await apiGet<Account>("/me");
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) return null;
    throw e;
  }
}

// For pages that need a login: sends visitors to /login and back afterwards.
export async function requireAccount(next: string): Promise<Account> {
  const account = await getAccount();
  if (!account) redirect(`/login?next=${encodeURIComponent(next)}`);
  return account;
}
