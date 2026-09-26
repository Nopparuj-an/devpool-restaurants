import "server-only";

import { notFound } from "next/navigation";

import { requireAccount } from "@/lib/api-server";
import type { AdminStatus } from "@/lib/types";

// Admin pages 404 for everyone else (the API also refuses, R-ADMIN-1).
export async function requireAdmin(path: string) {
  const account = await requireAccount(path);
  if (!account.is_admin) notFound();
  return account;
}

// Reads ?q=&status=&page= for the admin lists.
export function listParams(params: Record<string, string | string[] | undefined>) {
  const query = typeof params.q === "string" ? params.q.trim().slice(0, 100) : "";
  const status: AdminStatus = params.status === "active" || params.status === "banned" ? params.status : "";
  const page = Math.max(1, Math.floor(Number(params.page)) || 1);
  return { query, status, page };
}

export function apiQuery(p: { query: string; status: AdminStatus; page: number }, size: number) {
  const q = new URLSearchParams({ limit: String(size), offset: String((p.page - 1) * size) });
  if (p.query) q.set("q", p.query);
  if (p.status) q.set("status", p.status);
  return q;
}
