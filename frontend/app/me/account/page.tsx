import type { Metadata } from "next";

import { AccountPage } from "@/components/pages/pages";
import { requireAccount } from "@/lib/api-server";

export const metadata: Metadata = { title: "Account · Restaurants" };

export default async function Account() {
  const account = await requireAccount("/me/account");
  return <AccountPage account={account} />;
}
