import type { Metadata } from "next";

import { AdminUsersRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "Users · Admin" };

export default function Page() {
  return <AdminUsersRoute />;
}
