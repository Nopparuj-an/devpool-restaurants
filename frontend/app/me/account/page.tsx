import type { Metadata } from "next";

import { AccountRoute } from "@/app/me/account/container";

export const metadata: Metadata = { title: "Account · Restaurants" };

export default function Page() {
  return <AccountRoute />;
}
