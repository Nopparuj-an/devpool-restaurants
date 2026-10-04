"use client";

import { api } from "@/lib/api-client";
import { useRefresh, useRequireAccount } from "@/lib/queries";
import { gateLoggedIn } from "@/lib/route-helpers";

import { AccountScreen } from "./screen";

export function AccountRoute() {
  const refresh = useRefresh();
  const me = useRequireAccount();
  const wait = gateLoggedIn(me);
  if (wait) return wait;
  return (
    <AccountScreen
      account={me.data!}
      onSaveName={async (display_name) => {
        const res = await api("PUT", "/me", { display_name });
        if (!res.error) await refresh(); // header shows the new name
        return { error: res.error };
      }}
      onSetPassword={async (new_password) => {
        const res = await api("PUT", "/me/password", { new_password });
        if (!res.error) await refresh(); // has_password may have changed
        return { error: res.error };
      }}
    />
  );
}
