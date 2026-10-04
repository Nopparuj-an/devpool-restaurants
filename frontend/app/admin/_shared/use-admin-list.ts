"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect } from "react";

import { ADMIN_PAGE_SIZE, ADMIN_PAGE_SIZES } from "@/lib/paging";
import type { AdminStatus } from "@/lib/types";

// ?q=&status=&page=&size= for an admin list; changing search or status
// restarts at page 1. The selection lives in the route, so it survives all of
// these (R-ADMIN-8).
export function useAdminList(base: string) {
  const router = useRouter();
  const params = useSearchParams();
  const query = (params.get("q") ?? "").trim().slice(0, 100);
  const rawStatus = params.get("status");
  const status: AdminStatus = rawStatus === "active" || rawStatus === "banned" ? rawStatus : "";
  const page = Math.max(1, Math.floor(Number(params.get("page"))) || 1);
  const rawSize = Number(params.get("size"));
  const pageSize = (ADMIN_PAGE_SIZES as readonly number[]).includes(rawSize) ? rawSize : ADMIN_PAGE_SIZE;

  const apiParams = new URLSearchParams({ limit: String(pageSize), offset: String((page - 1) * pageSize) });
  if (query) apiParams.set("q", query);
  if (status) apiParams.set("status", status);

  const href = useCallback(
    (p: { query?: string; status?: AdminStatus; page?: number; size?: number }) => {
      const q = new URLSearchParams();
      const nextQuery = p.query ?? query;
      const nextStatus = p.status ?? status;
      const nextSize = p.size ?? pageSize;
      if (nextQuery) q.set("q", nextQuery);
      if (nextStatus) q.set("status", nextStatus);
      if (p.page && p.page > 1) q.set("page", String(p.page));
      if (nextSize !== ADMIN_PAGE_SIZE) q.set("size", String(nextSize));
      return q.size ? `${base}?${q}` : base;
    },
    [base, query, status, pageSize],
  );
  return {
    apiParams: String(apiParams),
    list: { query, status, page, pageSize },
    nav: {
      onQuery: useCallback((q: string) => router.replace(href({ query: q })), [router, href]),
      onStatus: (s: AdminStatus) => router.push(href({ status: s })),
      // Stay on the page that holds the first row shown now.
      onPageSize: (size: number) => router.push(href({ size, page: Math.floor(((page - 1) * pageSize) / size) + 1 })),
      hrefForPage: (p: number) => href({ page: p }),
    },
  };
}

// After a delete the current page can end up past the last one: go to the last.
export function usePageInRange(page: number, pageSize: number, total: number | undefined, hrefForPage: (p: number) => string) {
  const router = useRouter();
  const last = Math.max(1, Math.ceil((total ?? 0) / pageSize));
  useEffect(() => {
    if (total !== undefined && page > last) router.replace(hrefForPage(last));
  }, [total, page, last, router, hrefForPage]);
}
