import type { Metadata } from "next";

import { AuthRoute } from "@/app/_auth/container";

export const metadata: Metadata = { title: "Sign up · Restaurants" };

export default function Page() {
  return <AuthRoute mode="signup" />;
}
