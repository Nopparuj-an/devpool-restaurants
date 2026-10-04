"use client";

import { Search } from "lucide-react";
import { useEffect, useState } from "react";

import { RestaurantCard } from "@/components/app/restaurant-card";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import { Segmented } from "@/components/ui/controls";
import { Input } from "@/components/ui/field";
import { EmptyState } from "@/components/ui/misc";
import { LoadMore } from "@/components/ui/pagination";
import type { Account, RestaurantSummary, SortKey, SortOrder } from "@/lib/types";

const SORTS: { value: SortKey; label: string }[] = [
  { value: "top_rated", label: "Top rated" },
  { value: "most_reviewed", label: "Most reviewed" },
  { value: "newest", label: "New" },
];

// Search, sort and paging all happen on the server; the screen reports
// changes and the route turns them into ?q=&sort=. The list grows as you
// scroll (onMore loads the next page).
export function HomeScreen({
  account,
  restaurants,
  total,
  sort,
  order = "desc",
  query,
  onSort,
  onQuery,
  onMore,
  limitedIds = [],
}: {
  account: Account | null;
  restaurants: RestaurantSummary[];
  total: number;
  sort: SortKey;
  order?: SortOrder;
  query: string;
  // Clicking the active sort again flips the order; the route handles that.
  onSort: (sort: SortKey) => void;
  onQuery: (query: string) => void;
  onMore?: () => Promise<void>;
  limitedIds?: number[];
}) {
  const [text, setText] = useState(query);
  // Search 300 ms after typing stops, not on every key.
  useEffect(() => {
    if (text.trim() === query) return;
    const t = setTimeout(() => onQuery(text.trim()), 300);
    return () => clearTimeout(t);
  }, [text, query, onQuery]);

  return (
    <>
      <SiteHeader account={account} current="/" />
      <Page>
        <PageTitle title="Find a table" subtitle="Book a table at a local restaurant, or open your own." />
        <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative sm:w-72">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Search by name or cuisine"
              className="pl-9"
              aria-label="Search restaurants"
            />
          </div>
          <Segmented label="Sort by" options={SORTS} value={sort} onChange={onSort} direction={order} />
        </div>
        {query && (
          <p className="-mt-4 mb-6 text-sm text-muted">
            {total === 1 ? "1 restaurant" : `${total.toLocaleString("en")} restaurants`} match &quot;{query}&quot;
          </p>
        )}
        {restaurants.length === 0 ? (
          query ? (
            <EmptyState title="Nothing matches that search">Try a different name or cuisine.</EmptyState>
          ) : (
            <EmptyState title="No restaurants yet" />
          )
        ) : (
          <div className="grid gap-x-6 gap-y-10 sm:grid-cols-2 lg:grid-cols-3">
            {restaurants.map((r) => (
              <RestaurantCard key={r.id} restaurant={r} limited={limitedIds.includes(r.id)} />
            ))}
          </div>
        )}
        {onMore && <LoadMore shown={restaurants.length} total={total} onMore={onMore} />}
      </Page>
    </>
  );
}
