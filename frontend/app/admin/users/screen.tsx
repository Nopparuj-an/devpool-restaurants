"use client";

import { AdminFilters, AdminTabs, UserTable } from "@/components/app/admin";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import { EmptyState } from "@/components/ui/misc";
import type { AdminUser } from "@/lib/types";

import { useBan, useBulkDelete, type AdminListProps } from "@/app/admin/_shared/list-parts";

export function AdminUsersScreen({ users, ...p }: AdminListProps & { users: AdminUser[] }) {
  const ban = useBan(p.onBan);
  const bulk = useBulkDelete(p, "user");
  return (
    <>
      <SiteHeader account={p.account} current="/admin" />
      <Page>
        <PageTitle title="Admin" subtitle={`${p.total.toLocaleString("en")} ${p.total === 1 ? "user" : "users"}`} />
        <AdminTabs current="users" />
        <AdminFilters query={p.query} status={p.status} placeholder="Search by name or email" onQuery={p.onQuery} onStatus={p.onStatus} />
        {bulk.bar}
        {users.length === 0 ? (
          <EmptyState title="No users match" />
        ) : (
          <UserTable users={users} onBan={ban.open} selection={p.selection} selfId={p.account.id} />
        )}
        {bulk.footer}
        {ban.dialog}
      </Page>
    </>
  );
}
