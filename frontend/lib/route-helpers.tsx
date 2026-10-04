import type { UseQueryResult } from "@tanstack/react-query";
import { useParams } from "next/navigation";
import type { ReactNode } from "react";

import { NotFoundView, PageError, PageLoading } from "@/components/app/states";
import { ApiError } from "@/lib/api-client";
import type { Account } from "@/lib/types";

export type Loadable = { isLoading: boolean; isError: boolean; error: unknown; refetch: () => unknown };
export type Me = UseQueryResult<Account | null>;

// The screen to show instead of the page while anything is loading or failed,
// or null when everything is ready. A 404 shows the not-found page. Disabled
// queries (enabled: false) count as ready, with no data.
export function gate(me: Me, ...queries: Loadable[]): ReactNode {
  const all = [me, ...queries];
  const failed = all.find((q) => q.isError);
  if (failed) {
    if (failed.error instanceof ApiError && failed.error.status === 404) return <NotFoundView account={me.data} />;
    return <PageError account={me.data} onRetry={() => all.forEach((q) => q.isError && q.refetch())} />;
  }
  if (all.some((q) => q.isLoading)) return <PageLoading account={me.data} />;
  return null;
}

// Same, for pages behind a login: keeps showing "loading" while
// useRequireAccount redirects a logged-out visitor.
export function gateLoggedIn(me: Me, ...queries: Loadable[]): ReactNode {
  return gate(me, ...queries) ?? (me.data ? null : <PageLoading />);
}

// Offset paging for useInfiniteQuery: the next offset is how many rows are loaded.
export function nextOffset<P extends { total: number }>(rows: (p: P) => unknown[]) {
  return (last: P, all: P[]) => {
    const loaded = all.reduce((n, p) => n + rows(p).length, 0);
    return loaded < last.total ? loaded : undefined;
  };
}

// Rows can shift between pages when something is added meanwhile.
export function unique<T extends { id: number }>(rows: T[]): T[] {
  const seen = new Set<number>();
  return rows.filter((r) => !seen.has(r.id) && seen.add(r.id));
}

export function useIdParam() {
  return Number(useParams<{ id: string }>().id) || 0;
}
