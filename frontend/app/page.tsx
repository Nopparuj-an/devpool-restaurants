import { HomePage } from "@/components/pages/pages";
import { apiGet, getAccount } from "@/lib/api-server";
import type { RestaurantSummary, SortKey } from "@/lib/types";

const SORTS: SortKey[] = ["top_rated", "most_reviewed", "newest"];

export default async function Home({ searchParams }: PageProps<"/">) {
  const { sort: raw } = await searchParams;
  const sort = SORTS.includes(raw as SortKey) ? (raw as SortKey) : "top_rated";
  const [account, { restaurants }] = await Promise.all([
    getAccount(),
    apiGet<{ restaurants: RestaurantSummary[] }>(`/restaurants?sort=${sort}`),
  ]);
  return <HomePage account={account} restaurants={restaurants} sort={sort} />;
}
