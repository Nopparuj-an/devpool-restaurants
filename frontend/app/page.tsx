import { HomePage } from "@/components/pages/pages";
import { HOME_PAGE_SIZE } from "@/lib/paging";
import { apiGet, getAccount } from "@/lib/api-server";
import type { RestaurantPage, SortKey } from "@/lib/types";

const SORTS: SortKey[] = ["top_rated", "most_reviewed", "newest"];

// ?sort=&q=&page= are all handled by the API, so the list scales to any size.
export default async function Home({ searchParams }: PageProps<"/">) {
  const params = await searchParams;
  const sort = SORTS.includes(params.sort as SortKey) ? (params.sort as SortKey) : "top_rated";
  const query = typeof params.q === "string" ? params.q.trim().slice(0, 100) : "";
  const page = Math.max(1, Math.floor(Number(params.page)) || 1);

  const api = new URLSearchParams({ sort, limit: String(HOME_PAGE_SIZE), offset: String((page - 1) * HOME_PAGE_SIZE) });
  if (query) api.set("q", query);
  const [account, { restaurants, total }] = await Promise.all([
    getAccount(),
    apiGet<RestaurantPage>(`/restaurants?${api}`),
  ]);
  return <HomePage account={account} restaurants={restaurants} total={total} sort={sort} query={query} page={page} />;
}
