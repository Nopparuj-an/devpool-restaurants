import type { ReactNode } from "react";

import { NotFoundView } from "@/components/app/states";
import { gateLoggedIn, type Loadable, type Me } from "@/lib/route-helpers";

// Admin pages 404 for everyone else (the API also refuses, R-ADMIN-1).
export function gateAdmin(me: Me, ...queries: Loadable[]): ReactNode {
  const wait = gateLoggedIn(me, ...queries);
  if (wait) return wait;
  return me.data!.is_admin ? null : <NotFoundView account={me.data} />;
}
