import type { Metadata } from "next";

import { AuthRoute } from "@/app/_auth/container";

export const metadata: Metadata = { title: "Log in · Restaurants" };

export default function Page() {
  return <AuthRoute mode="login" />;
}
