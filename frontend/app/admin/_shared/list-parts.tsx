"use client";

import { useState } from "react";

import { BanDialog, DeleteDialog, SelectionBar, type BanTarget, type Selection } from "@/components/app/admin";
import { Pagination } from "@/components/ui/pagination";
import type { Account, ActionResult, AdminStatus } from "@/lib/types";

export type AdminListProps = {
  account: Account;
  total: number;
  query: string;
  status: AdminStatus;
  page: number;
  pageSize: number;
  onQuery: (q: string) => void;
  onStatus: (s: AdminStatus) => void;
  onPageSize: (size: number) => void;
  hrefForPage: (page: number) => string;
  onBan: (target: BanTarget, reason: string) => Promise<void>;
  // Bulk delete (R-ADMIN-8): the selection spans pages and filters.
  selection: Selection;
  onClearSelection: () => void;
  onDeleteSelected: () => ActionResult;
};

// The selection bar, the delete dialog, and the page links for an admin list.
export function useBulkDelete(p: AdminListProps, kind: "user" | "restaurant") {
  const [confirming, setConfirming] = useState(false);
  const noun: [string, string] = kind === "user" ? ["user", "users"] : ["restaurant", "restaurants"];
  return {
    bar: (
      <SelectionBar
        count={p.selection.selected.size}
        noun={noun}
        pageSize={p.pageSize}
        onPageSize={p.onPageSize}
        onClear={p.onClearSelection}
        onDelete={() => setConfirming(true)}
      />
    ),
    footer: (
      <>
        <Pagination page={p.page} pageCount={Math.ceil(p.total / p.pageSize)} hrefFor={p.hrefForPage} />
        <DeleteDialog
          kind={kind}
          names={[...p.selection.selected.values()]}
          open={confirming}
          onClose={() => setConfirming(false)}
          onConfirm={async () => {
            const res = await p.onDeleteSelected();
            if (!res.error) setConfirming(false);
            return res;
          }}
        />
      </>
    ),
  };
}

// One ban dialog per screen; tables only say which row was clicked.
export function useBan(onBan: (target: BanTarget, reason: string) => Promise<void>) {
  const [target, setTarget] = useState<BanTarget | null>(null);
  const dialog = (
    <BanDialog
      target={target}
      onClose={() => setTarget(null)}
      onConfirm={async (t, reason) => {
        await onBan(t, reason);
        setTarget(null);
      }}
    />
  );
  return { open: setTarget, dialog };
}
