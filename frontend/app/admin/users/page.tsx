import type { Metadata } from "next";

import { AdminUsersRoute } from "@/app/admin/users/container";

export const metadata: Metadata = { title: "Users · Admin" };

export default function Page() {
  return <AdminUsersRoute />;
}
