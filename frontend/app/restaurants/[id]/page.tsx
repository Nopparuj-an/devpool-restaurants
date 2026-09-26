import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { RestaurantPage } from "@/components/pages/pages";
import { apiGet, apiGetOrNull, getAccount } from "@/lib/api-server";
import { REVIEWS_PAGE_SIZE } from "@/lib/paging";
import type { Reservation, RestaurantDetail, Review, ReviewPage } from "@/lib/types";

export async function generateMetadata({ params }: PageProps<"/restaurants/[id]">): Promise<Metadata> {
  const { id } = await params;
  const r = await apiGetOrNull<RestaurantDetail>(`/restaurants/${Number(id) || 0}`);
  return { title: r ? `${r.name} · Restaurants` : "Restaurants" };
}

export default async function Restaurant({ params, searchParams }: PageProps<"/restaurants/[id]">) {
  const [{ id }, { edit }] = await Promise.all([params, searchParams]);
  const [account, restaurant] = await Promise.all([
    getAccount(),
    apiGetOrNull<RestaurantDetail>(`/restaurants/${Number(id) || 0}`),
  ]);
  if (!restaurant) notFound();

  const [{ reviews, total: reviewsTotal }, myReview, editing] = await Promise.all([
    apiGet<ReviewPage>(`/restaurants/${restaurant.id}/reviews?limit=${REVIEWS_PAGE_SIZE}`),
    account ? apiGetOrNull<Review>(`/restaurants/${restaurant.id}/reviews/me`) : null,
    // ?edit=<id> changes one of my bookings here (from My bookings).
    account && typeof edit === "string" ? apiGetOrNull<Reservation>(`/reservations/${Number(edit) || 0}`) : null,
  ]);

  return (
    <RestaurantPage
      account={account}
      restaurant={restaurant}
      reviews={reviews}
      reviewsTotal={reviewsTotal}
      myReview={myReview ?? undefined}
      editing={editing?.restaurant.id === restaurant.id && editing.can_modify ? editing : undefined}
    />
  );
}
