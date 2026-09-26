"use client";

import { Search } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { ConfirmDialog, Segmented } from "@/components/ui/controls";
import { Input, Textarea } from "@/components/ui/field";
import { Badge, Rating } from "@/components/ui/misc";
import * as fmt from "@/lib/format";
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

export function UserTable({ users, onBan }: { users: AdminUser[]; onBan: (t: BanTarget) => void }) {
  const tz = useTimeZone();
  return (
    <div className="overflow-x-auto rounded-xl border border-line">
      <table className="w-full min-w-[48rem] text-left text-sm">
        <thead className="border-b border-line bg-surface text-muted">
          <tr>
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
            <tr key={u.id} className="border-b border-line last:border-0">
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
                {u.ban_reason && <div className="mt-1 max-w-48 truncate text-xs text-muted" title={u.ban_reason}>{u.ban_reason}</div>}
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
}: {
  restaurants: AdminRestaurant[];
  onBan: (t: BanTarget) => void;
  showOwner?: boolean;
}) {
  return (
    <div className="overflow-x-auto rounded-xl border border-line">
      <table className="w-full min-w-[44rem] text-left text-sm">
        <thead className="border-b border-line bg-surface text-muted">
          <tr>
            <th className="px-4 py-2.5 font-medium">Restaurant</th>
            {showOwner && <th className="px-4 py-2.5 font-medium">Owner</th>}
            <th className="px-4 py-2.5 font-medium">Rating</th>
            <th className="px-4 py-2.5 font-medium">Status</th>
            <th className="px-4 py-2.5" />
          </tr>
        </thead>
        <tbody>
          {restaurants.map((r) => (
            <tr key={r.id} className="border-b border-line last:border-0">
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
                {r.ban_reason && <div className="mt-1 max-w-48 truncate text-xs text-muted" title={r.ban_reason}>{r.ban_reason}</div>}
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
