"use client";

import { useCallback } from "react";

import type { BanTarget } from "@/components/app/admin";
import { api } from "@/lib/api-client";
import { useRefresh } from "@/lib/queries";

// Calls the admin ban / unban endpoint for a user or restaurant, then refreshes.
export function useBanAction() {
  const refresh = useRefresh();
  return useCallback(
    async (t: BanTarget, reason: string) => {
      const base = t.kind === "user" ? `/admin/users/${t.id}` : `/admin/restaurants/${t.id}`;
      const res = t.banned ? await api("POST", `${base}/unban`) : await api("POST", `${base}/ban`, { reason });
      if (res.error) window.alert(res.error);
      await refresh();
    },
    [refresh],
  );
}
