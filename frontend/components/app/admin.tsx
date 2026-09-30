"use client";

import { Search, Trash2 } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { ConfirmDialog, Segmented } from "@/components/ui/controls";
import { Field, Input, Select, Textarea } from "@/components/ui/field";
import { Badge, Notice, Rating } from "@/components/ui/misc";
import * as fmt from "@/lib/format";
import { ADMIN_PAGE_SIZES } from "@/lib/paging";
import type { AdminRestaurant, AdminStatus, AdminUser } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

export function AdminTabs({ current }: { current: "users" | "restaurants" }) {
  const tab = (key: typeof current, href: string, label: string) => (
    <Link
      href={href}
      aria-current={current === key ? "page" : undefined}
      className="border-b-2 border-transparent px-1 pb-3 text-sm text-muted hover:text-ink aria-[current=page]:border-accent aria-[current=page]:font-medium aria-[current=page]:text-ink"
    >
      {label}
    </Link>
  );
  return (
    <nav className="mb-8 flex gap-6 border-b border-line">
      {tab("users", "/admin/users", "Users")}
      {tab("restaurants", "/admin/restaurants", "Restaurants")}
    </nav>
  );
}

// Search box (debounced) plus All / Active / Banned.
export function AdminFilters({
  query,
  status,
  placeholder,
  onQuery,
  onStatus,
}: {
  query: string;
  status: AdminStatus;
  placeholder: string;
  onQuery: (q: string) => void;
  onStatus: (s: AdminStatus) => void;
}) {
  const [text, setText] = useState(query);
  useEffect(() => {
    if (text.trim() === query) return;
    const t = setTimeout(() => onQuery(text.trim()), 300);
    return () => clearTimeout(t);
  }, [text, query, onQuery]);
  return (
    <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="relative sm:w-80">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-faint" />
        <Input value={text} onChange={(e) => setText(e.target.value)} placeholder={placeholder} className="pl-9" aria-label="Search" />
      </div>
      <Segmented
        label="Status"
        value={status || "all"}
        onChange={(v) => onStatus(v === "all" ? "" : (v as AdminStatus))}
        options={[
          { value: "all", label: "All" },
          { value: "active", label: "Active" },
          { value: "banned", label: "Banned" },
        ]}
      />
    </div>
  );
}

// Bulk selection (R-ADMIN-8). The route keeps it across pages and filters,
// keyed by id with the name for the confirm dialog.
export type Selection = {
  selected: ReadonlyMap<number, string>;
  onSelect: (rows: { id: number; name: string }[], on: boolean) => void;
};

function Checkbox({
  checked,
  indeterminate = false,
  disabled,
  label,
  onChange,
}: {
  checked: boolean;
  indeterminate?: boolean;
  disabled?: boolean;
  label: string;
  onChange: (on: boolean) => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = indeterminate;
  }, [indeterminate]);
  return (
    <input
      ref={ref}
      type="checkbox"
      aria-label={label}
      checked={checked}
      disabled={disabled}
      onChange={(e) => onChange(e.target.checked)}
      className="size-4 accent-accent disabled:opacity-30"
    />
  );
}

// "Select all" for the rows on this page; rows elsewhere stay as they are.
function SelectAll({ rows, selection }: { rows: { id: number; name: string }[]; selection: Selection }) {
  const on = rows.filter((r) => selection.selected.has(r.id)).length;
  return (
    <Checkbox
      label="Select all on this page"
      checked={rows.length > 0 && on === rows.length}
      indeterminate={on > 0 && on < rows.length}
      disabled={rows.length === 0}
      onChange={(v) => selection.onSelect(rows, v)}
    />
  );
}

// Above an admin table: what's selected (on any page), and rows per page.
export function SelectionBar({
  count,
  noun,
  pageSize,
  onPageSize,
  onClear,
  onDelete,
}: {
  count: number;
  noun: [string, string];
  pageSize: number;
  onPageSize: (size: number) => void;
  onClear: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="mb-3 flex min-h-9 flex-wrap items-center justify-between gap-3 text-sm">
      {count > 0 ? (
        <span className="flex flex-wrap items-center gap-3">
          <span className="font-medium tabular-nums">
            {count.toLocaleString("en")} {count === 1 ? noun[0] : noun[1]} selected
          </span>
          <Button size="sm" variant="ghost" onClick={onClear}>
            Clear
          </Button>
          <Button size="sm" variant="danger" onClick={onDelete}>
            <Trash2 className="size-4" /> Delete
          </Button>
        </span>
      ) : (
        <span className="text-muted">Tick rows to delete them. The selection stays when you change page.</span>
      )}
      <label className="flex items-center gap-2 text-muted">
        Per page
        <Select value={pageSize} onChange={(e) => onPageSize(Number(e.target.value))} className="w-24">
          {ADMIN_PAGE_SIZES.map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </Select>
      </label>
    </div>
  );
}

// Deleting can't be undone, so the dialog names what goes with it.
export function DeleteDialog({
  kind,
  names,
  open,
  onClose,
  onConfirm,
}: {
  kind: "user" | "restaurant";
  names: string[];
  open: boolean;
  onClose: () => void;
  onConfirm: () => Promise<{ error?: string }>;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const n = names.length;
  const noun = kind === "user" ? (n === 1 ? "user" : "users") : n === 1 ? "restaurant" : "restaurants";
  const shown = names.slice(0, 5).join(", ") + (n > 5 ? ` and ${(n - 5).toLocaleString("en")} more` : "");
  return (
    <ConfirmDialog
      open={open}
      title={`Delete ${n === 1 ? names[0] : `${n.toLocaleString("en")} ${noun}`}?`}
      confirmLabel={busy ? "Deleting…" : `Delete ${noun}`}
      danger
      onClose={() => {
        setError(undefined);
        onClose();
      }}
      onConfirm={async () => {
        if (busy) return;
        setBusy(true);
        const res = await onConfirm();
        setBusy(false);
        setError(res.error);
      }}
    >
      <div className="flex flex-col gap-3">
        {n > 1 && <p className="text-ink">{shown}</p>}
        <p>
          {kind === "user"
            ? "Their restaurants go too, with every booking, review and photo at them, and so do their own bookings and reviews. Ratings they counted in are recalculated."
            : "Their bookings, reviews and photos go too."}{" "}
          This can&apos;t be undone. To hide {n === 1 ? "it" : "them"} for now, ban instead.
        </p>
        {error && <Notice tone="danger">{error}</Notice>}
      </div>
    </ConfirmDialog>
  );
}

export function StatusBadge({ bannedAt, admin }: { bannedAt: string | null; admin?: boolean }) {
  if (bannedAt) return <Badge tone="danger">Banned</Badge>;
  if (admin) return <Badge tone="accent">Admin</Badge>;
  return <Badge tone="success">Active</Badge>;
}

export type BanTarget = { kind: "user" | "restaurant"; id: number; name: string; banned: boolean };

// Ban asks for an optional reason and says what will happen; unban just confirms.
export function BanDialog({
  target,
  onClose,
  onConfirm,
}: {
  target: BanTarget | null;
  onClose: () => void;
  onConfirm: (target: BanTarget, reason: string) => Promise<void>;
}) {
  const [reason, setReason] = useState("");
  const [shownTarget, setShownTarget] = useState(target);
  if (target !== shownTarget) {
    setShownTarget(target);
    setReason("");
  }
  if (!target) return null;
  const ban = !target.banned;
  const what =
    target.kind === "user"
      ? "They can't log in, and their restaurants and reviews are hidden. Ratings they affected are recalculated without their reviews."
      : "Customers can't find, book or review it. Existing bookings stay, and customers can still cancel them.";
  return (
    <ConfirmDialog
      open
      title={`${ban ? "Ban" : "Unban"} ${target.name}?`}
      confirmLabel={ban ? "Ban" : "Unban"}
      danger={ban}
      onClose={onClose}
      onConfirm={() => onConfirm(target, reason)}
    >
      {ban ? (
        <div className="flex flex-col gap-3">
          <p>{what} You can undo this later.</p>
          <label className="flex flex-col gap-1.5">
            <span className="font-medium text-ink">Reason (optional)</span>
            <Textarea value={reason} onChange={(e) => setReason(e.target.value)} maxLength={500} className="min-h-20" />
          </label>
        </div>
      ) : (
        <p>Everything comes back as it was{target.kind === "user" ? ", including their reviews in ratings" : ""}.</p>
      )}
    </ConfirmDialog>
  );
}

// An admin renames a user (R-ADMIN-6). The email is the login and can't change.
export function NameForm({ name: current, onSave }: { name: string; onSave: (name: string) => Promise<{ error?: string }> }) {
  const [name, setName] = useState(current);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const changed = name.trim() !== current && name.trim() !== "";
  return (
    <form
      className="flex max-w-md flex-col gap-3"
      onSubmit={async (e) => {
        e.preventDefault();
        setBusy(true);
        const res = await onSave(name.trim());
        setBusy(false);
        setError(res.error);
      }}
    >
      <Field label="Display name" hint="Shown on their reviews and profile.">
        <div className="flex gap-2">
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} required />
          <Button type="submit" variant="secondary" disabled={!changed || busy}>
            {busy ? "Saving…" : "Rename"}
          </Button>
        </div>
      </Field>
      {error && <Notice tone="danger">{error}</Notice>}
    </form>
  );
}

// Admins and yourself can't be deleted (R-ADMIN-8), so they have no checkbox.
export function UserTable({
  users,
  onBan,
  selection,
  selfId,
}: {
  users: AdminUser[];
  onBan: (t: BanTarget) => void;
  selection?: Selection;
  selfId?: number;
}) {
  const tz = useTimeZone();
  const selectable = users.filter((u) => !u.is_admin && u.id !== selfId).map((u) => ({ id: u.id, name: u.display_name }));
  return (
    <div className="overflow-x-auto rounded-xl border border-line">
      <table className="w-full min-w-[48rem] text-left text-sm">
        <thead className="border-b border-line bg-surface text-muted">
          <tr>
            {selection && (
              <th className="w-10 py-2.5 pl-4">
                <SelectAll rows={selectable} selection={selection} />
              </th>
            )}
            <th className="px-4 py-2.5 font-medium">Name</th>
            <th className="px-4 py-2.5 font-medium">Restaurants</th>
            <th className="px-4 py-2.5 font-medium">Reviews</th>
            <th className="px-4 py-2.5 font-medium">Joined</th>
            <th className="px-4 py-2.5 font-medium">Status</th>
            <th className="px-4 py-2.5" />
          </tr>
        </thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.id} className={`border-b border-line last:border-0 ${selection?.selected.has(u.id) ? "bg-accent-soft" : ""}`}>
              {selection && (
                <td className="py-3 pl-4">
                  {!u.is_admin && u.id !== selfId && (
                    <Checkbox
                      label={`Select ${u.display_name}`}
                      checked={selection.selected.has(u.id)}
                      onChange={(on) => selection.onSelect([{ id: u.id, name: u.display_name }], on)}
                    />
                  )}
                </td>
              )}
              <td className="px-4 py-3">
                <Link href={`/admin/users/${u.id}`} className="font-medium hover:text-accent">
                  {u.display_name}
                </Link>
                <div className="text-muted">{u.email}</div>
              </td>
              <td className="px-4 py-3 tabular-nums">{u.restaurants}</td>
              <td className="px-4 py-3 tabular-nums">{u.reviews}</td>
              <td className="px-4 py-3 text-muted">{fmt.day(u.created_at, tz)}</td>
              <td className="px-4 py-3">
                <StatusBadge bannedAt={u.banned_at} admin={u.is_admin} />
                {u.ban_reason && (
                  <div className="mt-1 max-w-48 truncate text-xs text-muted" title={u.ban_reason}>
                    {u.ban_reason}
                  </div>
                )}
              </td>
              <td className="px-4 py-3 text-right">
                {!u.is_admin && (
                  <Button
                    size="sm"
                    variant={u.banned_at ? "secondary" : "danger"}
                    onClick={() => onBan({ kind: "user", id: u.id, name: u.display_name, banned: !!u.banned_at })}
                  >
                    {u.banned_at ? "Unban" : "Ban"}
                  </Button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function RestaurantTable({
  restaurants,
  onBan,
  showOwner = true,
  selection,
}: {
  restaurants: AdminRestaurant[];
  onBan: (t: BanTarget) => void;
  showOwner?: boolean;
  selection?: Selection;
}) {
  return (
    <div className="overflow-x-auto rounded-xl border border-line">
      <table className="w-full min-w-[44rem] text-left text-sm">
        <thead className="border-b border-line bg-surface text-muted">
          <tr>
            {selection && (
              <th className="w-10 py-2.5 pl-4">
                <SelectAll rows={restaurants.map((r) => ({ id: r.id, name: r.name }))} selection={selection} />
              </th>
            )}
            <th className="px-4 py-2.5 font-medium">Restaurant</th>
            {showOwner && <th className="px-4 py-2.5 font-medium">Owner</th>}
            <th className="px-4 py-2.5 font-medium">Rating</th>
            <th className="px-4 py-2.5 font-medium">Status</th>
            <th className="px-4 py-2.5" />
          </tr>
        </thead>
        <tbody>
          {restaurants.map((r) => (
            <tr key={r.id} className={`border-b border-line last:border-0 ${selection?.selected.has(r.id) ? "bg-accent-soft" : ""}`}>
              {selection && (
                <td className="py-3 pl-4">
                  <Checkbox
                    label={`Select ${r.name}`}
                    checked={selection.selected.has(r.id)}
                    onChange={(on) => selection.onSelect([{ id: r.id, name: r.name }], on)}
                  />
                </td>
              )}
              <td className="px-4 py-3">
                <Link href={`/restaurants/${r.id}`} className="font-medium hover:text-accent">
                  {r.name}
                </Link>
                <div className="text-muted">
                  {r.cuisine} · {r.location}
                </div>
              </td>
              {showOwner && (
                <td className="px-4 py-3">
                  <Link href={`/admin/users/${r.owner.id}`} className="hover:text-accent">
                    {r.owner.display_name}
                  </Link>
                  {r.owner.banned && <div className="text-xs text-danger">Owner banned</div>}
                </td>
              )}
              <td className="px-4 py-3">
                <Rating value={r.rating} count={r.review_count} />
              </td>
              <td className="px-4 py-3">
                <StatusBadge bannedAt={r.banned_at} />
                {r.ban_reason && (
                  <div className="mt-1 max-w-48 truncate text-xs text-muted" title={r.ban_reason}>
                    {r.ban_reason}
                  </div>
                )}
              </td>
              <td className="px-4 py-3 text-right">
                <Button
                  size="sm"
                  variant={r.banned_at ? "secondary" : "danger"}
                  onClick={() => onBan({ kind: "restaurant", id: r.id, name: r.name, banned: !!r.banned_at })}
                >
                  {r.banned_at ? "Unban" : "Ban"}
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
