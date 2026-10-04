"use client";

import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback } from "react";

import { get } from "@/lib/api-client";
import { HOME_PAGE_SIZE } from "@/lib/paging";
import { keys, useAccount } from "@/lib/queries";
import { gate, nextOffset, unique } from "@/lib/route-helpers";
import type { RestaurantPage, SortKey, SortOrder } from "@/lib/types";

import { HomeScreen } from "./screen";

const SORTS: SortKey[] = ["top_rated", "most_reviewed", "newest"];

// ?sort=&q= are handled by the API, so the list scales to any size. It grows
// as you scroll, HOME_PAGE_SIZE at a time.
export function HomeRoute() {
  const router = useRouter();
  const params = useSearchParams();
  const sort = SORTS.includes(params.get("sort") as SortKey) ? (params.get("sort") as SortKey) : "top_rated";
  const order: SortOrder = params.get("order") === "asc" ? "asc" : "desc";
  const query = (params.get("q") ?? "").trim().slice(0, 100);

  const apiParams = new URLSearchParams({ sort, order, limit: String(HOME_PAGE_SIZE) });
  if (query) apiParams.set("q", query);
  const me = useAccount();
  const list = useInfiniteQuery({
    queryKey: keys.restaurants(String(apiParams)),
    queryFn: ({ pageParam }) => get<RestaurantPage>(`/restaurants?${apiParams}&offset=${pageParam}`),
    initialPageParam: 0,
    getNextPageParam: nextOffset<RestaurantPage>((p) => p.restaurants),
    // Keep the current list on screen while a new search or sort loads.
    placeholderData: keepPreviousData,
  });

  const href = useCallback(
    (p: { sort?: SortKey; order?: SortOrder; query?: string }) => {
      const q = new URLSearchParams();
      const nextSort = p.sort ?? sort;
      const nextOrder = p.order ?? order;
      const nextQuery = p.query ?? query;
      if (nextSort !== "top_rated") q.set("sort", nextSort);
      if (nextOrder === "asc") q.set("order", nextOrder);
      if (nextQuery) q.set("q", nextQuery);
      return q.size ? `/?${q}` : "/";
    },
    [sort, order, query],
  );
  const onQuery = useCallback((q: string) => router.replace(href({ query: q })), [router, href]);

  const wait = gate(me, list);
  if (wait) return wait;
  const pages = list.data!.pages;
  return (
    <HomeScreen
      account={me.data ?? null}
      restaurants={unique(pages.flatMap((p) => p.restaurants))}
      total={pages[0].total}
      sort={sort}
      order={order}
      query={query}
      // Picking another sort resets to the default direction; the same one flips it.
      onSort={(s) => router.push(href({ sort: s, order: s === sort && order === "desc" ? "asc" : "desc" }))}
      onQuery={onQuery}
      onMore={async () => {
        await list.fetchNextPage();
      }}
    />
  );
}
