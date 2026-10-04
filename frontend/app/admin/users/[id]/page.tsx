import type { Metadata } from "next";

import { AdminUserRoute } from "@/app/admin/users/[id]/container";

export const metadata: Metadata = { title: "User · Admin" };

export default function Page() {
  return <AdminUserRoute />;
}
