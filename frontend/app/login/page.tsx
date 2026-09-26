import type { Metadata } from "next";

import { AuthRoute } from "./auth-route";

export const metadata: Metadata = { title: "Log in · Restaurants" };

export default function Login({ searchParams }: PageProps<"/login">) {
  return <AuthRoute mode="login" searchParams={searchParams} />;
}
