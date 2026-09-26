import type { Metadata } from "next";

import { RestaurantEditorPage } from "@/components/pages/pages";

import { ownedRestaurant } from "../owned";

export const metadata: Metadata = { title: "Edit restaurant · Restaurants" };

export default async function EditRestaurant({ params }: PageProps<"/me/restaurants/[id]/edit">) {
  const { id } = await params;
  const { account, restaurant } = await ownedRestaurant(id, `/me/restaurants/${id}/edit`);
  return <RestaurantEditorPage account={account} restaurant={restaurant} />;
}
