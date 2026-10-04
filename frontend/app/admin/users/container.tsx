"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { get } from "@/lib/api-client";
import { keys, useRequireAccount } from "@/lib/queries";
import type { AdminUser } from "@/lib/types";

import { gateAdmin } from "@/app/admin/_shared/gate-admin";
import { useAdminList, usePageInRange } from "@/app/admin/_shared/use-admin-list";
import { useBanAction } from "@/lib/use-ban-action";
import { useSelection } from "@/app/admin/_shared/use-selection";
import { AdminUsersScreen } from "./screen";

export function AdminUsersRoute() {
  const me = useRequireAccount();
  const { apiParams, list, nav } = useAdminList("/admin/users");
  const users = useQuery({
    queryKey: keys.adminUsers(apiParams),
    queryFn: () => get<{ users: AdminUser[]; total: number }>(`/admin/users?${apiParams}`),
    enabled: !!me.data?.is_admin,
    placeholderData: keepPreviousData,
  });
  const ban = useBanAction();
  const bulk = useSelection("users");
  usePageInRange(list.page, list.pageSize, users.data?.total, nav.hrefForPage);
  const wait = gateAdmin(me, users);
  if (wait) return wait;
  return (
    <AdminUsersScreen account={me.data!} {...list} {...nav} {...bulk} users={users.data!.users} total={users.data!.total} onBan={ban} />
  );
}
