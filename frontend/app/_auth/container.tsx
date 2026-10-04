"use client";

import { useQuery } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";

import type { AuthInput } from "@/components/app/auth-form";
import { PageLoading } from "@/components/app/states";
import { api, get } from "@/lib/api-client";
import { keys, useAccount, useRefresh } from "@/lib/queries";
import { gate } from "@/lib/route-helpers";

import { AuthScreen } from "./screen";

// /login and /signup. Only same-site paths are allowed for ?next=.
export function AuthRoute({ mode }: { mode: "login" | "signup" }) {
  const router = useRouter();
  const refresh = useRefresh();
  const params = useSearchParams();
  const rawNext = params.get("next");
  const next = rawNext && rawNext.startsWith("/") && !rawNext.startsWith("//") ? rawNext : "/";
  const me = useAccount();
  const providers = useQuery({
    queryKey: keys.providers,
    queryFn: () => get<{ google: boolean }>("/auth/providers"),
    staleTime: Infinity,
  });

  // Already logged in, or just logged in: go on to ?next=.
  const loggedIn = !!me.data;
  useEffect(() => {
    if (loggedIn) router.replace(next);
  }, [loggedIn, next, router]);

  async function onSubmit(input: AuthInput) {
    const res = await api("POST", mode === "login" ? "/auth/login" : "/auth/signup", input);
    if (res.error) return { error: res.error };
    // Everything cached so far was loaded logged out. Refetching /me also
    // triggers the redirect above.
    await refresh();
    return {};
  }

  const wait = gate(me, providers);
  if (wait) return wait;
  if (loggedIn) return <PageLoading account={me.data} />;
  return (
    <AuthScreen
      mode={mode}
      googleEnabled={providers.data!.google}
      error={params.get("error") ?? undefined}
      next={next}
      onSubmit={onSubmit}
    />
  );
}
