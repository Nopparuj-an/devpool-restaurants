import type { Metadata } from "next";

import { OwnerBookingsPage } from "@/components/pages/pages";

import { ownedRestaurant } from "../owned";

export const metadata: Metadata = { title: "Bookings · Restaurants" };

export default async function RestaurantBookings({ params }: PageProps<"/me/restaurants/[id]/bookings">) {
  const { id } = await params;
  const { account, restaurant } = await ownedRestaurant(id, `/me/restaurants/${id}/bookings`);
  return <OwnerBookingsPage account={account} restaurant={restaurant} />;
}
