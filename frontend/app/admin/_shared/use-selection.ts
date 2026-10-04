"use client";

import { useCallback, useState } from "react";

import type { Selection } from "@/components/app/admin";
import { api } from "@/lib/api-client";
import { ADMIN_DELETE_CHUNK } from "@/lib/paging";
import { useRefresh } from "@/lib/queries";

// Rows ticked on any page, by id, with the name for the confirm dialog.
export function useSelection(kind: "users" | "restaurants") {
  const refresh = useRefresh();
  const [selected, setSelected] = useState<ReadonlyMap<number, string>>(() => new Map());
  const onSelect = useCallback((rows: { id: number; name: string }[], on: boolean) => {
    setSelected((prev) => {
      const next = new Map(prev);
      for (const r of rows) {
        if (on) next.set(r.id, r.name);
        else next.delete(r.id);
      }
      return next;
    });
  }, []);
  const selection: Selection = { selected, onSelect };

  // The API takes ADMIN_DELETE_CHUNK ids per request. Chunks that went
  // through leave the selection even if a later one fails.
  async function onDeleteSelected() {
    const ids = [...selected.keys()];
    let error: string | undefined;
    for (let i = 0; i < ids.length && !error; i += ADMIN_DELETE_CHUNK) {
      const chunk = ids.slice(i, i + ADMIN_DELETE_CHUNK);
      const res = await api("POST", `/admin/${kind}/delete`, { ids: chunk });
      if (res.error) error = res.error;
      else
        onSelect(
          chunk.map((id) => ({ id, name: "" })),
          false,
        );
    }
    await refresh();
    return { error };
  }
  return { selection, onClearSelection: () => setSelected(new Map()), onDeleteSelected };
}
