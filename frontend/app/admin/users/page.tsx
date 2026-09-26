import type { Metadata } from "next";

import { AdminUsersPage } from "@/components/pages/pages";
import { apiGet } from "@/lib/api-server";
import { ADMIN_PAGE_SIZE } from "@/lib/paging";
import type { AdminUser } from "@/lib/types";

import { apiQuery, listParams, requireAdmin } from "../guard";

export const metadata: Metadata = { title: "Users · Admin" };

export default async function AdminUsers({ searchParams }: PageProps<"/admin/users">) {
  const account = await requireAdmin("/admin/users");
  const p = listParams(await searchParams);
  const { users, total } = await apiGet<{ users: AdminUser[]; total: number }>(`/admin/users?${apiQuery(p, ADMIN_PAGE_SIZE)}`);
  return <AdminUsersPage account={account} users={users} total={total} {...p} />;
}
