import type { Metadata } from "next";

import { AuthRoute } from "../login/auth-route";

export const metadata: Metadata = { title: "Sign up · Restaurants" };

export default function Signup({ searchParams }: PageProps<"/signup">) {
  return <AuthRoute mode="signup" searchParams={searchParams} />;
}
