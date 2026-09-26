"use client";

// TanStack Query setup and the hooks every route shares. Pages read with
// useQuery; after any write they call useRefresh(), which marks every cached
// read stale so whatever is on screen refetches (like a page reload, minus
// the reload).
import { QueryCache, QueryClient, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useCallback, useEffect } from "react";

import { ApiError, get } from "./api-client";
import type { Account } from "./types";

export function makeQueryClient() {
  const client: QueryClient = new QueryClient({
    // A 401 on any read means the session ended: show the app logged out.
    queryCache: new QueryCache({
      onError: (e) => {
        if (e instanceof ApiError && e.status === 401) client.setQueryData(keys.me, null);
      },
    }),
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        // 4xx answers won't change on a retry; network and 5xx errors might.
        retry: (count, e) => !(e instanceof ApiError && e.status < 500) && count < 2,
      },
    },
  });
  return client;
}

// Keys are nested so a restaurant's reviews, availability and bookings sit
// under ["restaurant", id].
export const keys = {
  me: ["me"] as const,
  providers: ["auth", "providers"] as const,
  restaurants: (params: string) => ["restaurants", params] as const,
  restaurant: (id: number) => ["restaurant", id] as const,
  reviews: (id: number) => ["restaurant", id, "reviews"] as const,
  myReview: (id: number) => ["restaurant", id, "my-review"] as const,
  ownerDay: (id: number, from: string, to: string) => ["restaurant", id, "owner-day", from, to] as const,
  reservation: (id: number) => ["reservation", id] as const,
  myReservations: ["me", "reservations"] as const,
  myRestaurants: ["me", "restaurants"] as const,
  adminUsers: (params: string) => ["admin", "users", params] as const,
  adminUser: (id: number) => ["admin", "user", id] as const,
  adminRestaurants: (params: string) => ["admin", "restaurants", params] as const,
  profile: (id: number) => ["user", id] as const,
  profileRestaurants: (id: number) => ["user", id, "restaurants"] as const,
  profileReviews: (id: number) => ["user", id, "reviews"] as const,
};

// The logged-in account: undefined while loading, null when logged out.
export function useAccount() {
  return useQuery({
    queryKey: keys.me,
    queryFn: async () => {
      try {
        return await get<Account>("/me");
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null;
        throw e;
      }
    },
    staleTime: 5 * 60_000,
  });
}

// For pages that need a login: sends visitors to /login and back afterwards.
// The API refuses the data anyway; this only picks the right screen.
export function useRequireAccount() {
  const me = useAccount();
  const router = useRouter();
  useEffect(() => {
    if (me.data !== null) return;
    const here = window.location.pathname + window.location.search;
    router.replace(`/login?next=${encodeURIComponent(here)}`);
  }, [me.data, router]);
  return me;
}

export function useRefresh() {
  const client = useQueryClient();
  return useCallback(() => client.invalidateQueries(), [client]);
}
