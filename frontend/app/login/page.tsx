import type { Metadata } from "next";

import { AuthRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "Log in · Restaurants" };

export default function Page() {
  return <AuthRoute mode="login" />;
}
