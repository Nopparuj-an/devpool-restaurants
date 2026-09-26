import type { Metadata } from "next";

import { AuthRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "Sign up · Restaurants" };

export default function Page() {
  return <AuthRoute mode="signup" />;
}
