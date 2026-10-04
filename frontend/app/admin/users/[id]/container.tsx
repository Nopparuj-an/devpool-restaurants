"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";

import { api, get } from "@/lib/api-client";
import { keys, useRefresh, useRequireAccount } from "@/lib/queries";
import { useIdParam } from "@/lib/route-helpers";
import type { AdminUserDetail } from "@/lib/types";

import { gateAdmin } from "@/app/admin/_shared/gate-admin";
import { useBanAction } from "@/lib/use-ban-action";
import { AdminUserScreen } from "./screen";

export function AdminUserRoute() {
  const id = useIdParam();
  const me = useRequireAccount();
  const user = useQuery({
    queryKey: keys.adminUser(id),
    queryFn: () => get<AdminUserDetail>(`/admin/users/${id}`),
    enabled: !!me.data?.is_admin,
  });
  const ban = useBanAction();
  const refresh = useRefresh();
  const client = useQueryClient();
  const router = useRouter();
  const wait = gateAdmin(me, user);
  if (wait) return wait;
  return (
    <AdminUserScreen
      account={me.data!}
      user={user.data!}
      onBan={ban}
      onRename={async (display_name) => {
        const res = await api("PUT", `/admin/users/${id}`, { display_name });
        if (!res.error) await refresh();
        return { error: res.error };
      }}
      // Swaps the session cookie for one as this user (R-ADMIN-7). Nothing
      // cached as the admin is valid any more.
      onImpersonate={async () => {
        const res = await api("POST", `/admin/users/${id}/impersonate`);
        if (res.error) return { error: res.error };
        client.resetQueries();
        router.push("/");
        return {};
      }}
    />
  );
}
