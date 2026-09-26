import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { AdminUserPage } from "@/components/pages/pages";
import { apiGetOrNull } from "@/lib/api-server";
import type { AdminUserDetail } from "@/lib/types";

import { requireAdmin } from "../../guard";

export const metadata: Metadata = { title: "User · Admin" };

export default async function AdminUser({ params }: PageProps<"/admin/users/[id]">) {
  const { id } = await params;
  const account = await requireAdmin(`/admin/users/${id}`);
  const user = await apiGetOrNull<AdminUserDetail>(`/admin/users/${Number(id) || 0}`);
  if (!user) notFound();
  return <AdminUserPage account={account} user={user} />;
}
