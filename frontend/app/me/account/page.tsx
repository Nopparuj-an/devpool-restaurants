import type { Metadata } from "next";

import { AccountRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "Account · Restaurants" };

export default function Page() {
  return <AccountRoute />;
}
