import type { Metadata } from "next";

import { AdminUserRoute } from "@/components/pages/pages";

export const metadata: Metadata = { title: "User · Admin" };

export default function Page() {
  return <AdminUserRoute />;
}
