"use client";

import { VenetianMask } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { NameForm, RestaurantTable, StatusBadge, type BanTarget } from "@/components/app/admin";
import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import { Button, ButtonLink } from "@/components/ui/button";
import { EmptyState, Notice } from "@/components/ui/misc";
import * as fmt from "@/lib/format";
import type { Account, ActionResult, AdminUserDetail } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

import { useBan } from "@/app/admin/_shared/list-parts";

export function AdminUserScreen({
  account,
  user,
  onBan,
  onRename,
  onImpersonate,
}: {
  account: Account;
  user: AdminUserDetail;
  onBan: (target: BanTarget, reason: string) => Promise<void>;
  onRename?: (name: string) => ActionResult;
  onImpersonate?: () => ActionResult;
}) {
  const ban = useBan(onBan);
  const [impersonating, setImpersonating] = useState(false);
  const [impersonateError, setImpersonateError] = useState<string>();
  // Admins and banned users can't be impersonated (R-ADMIN-7).
  const canImpersonate = onImpersonate && !user.is_admin && !user.banned_at && user.id !== account.id;
  const tz = useTimeZone();
  const stats = [
    ["Restaurants", user.restaurants],
    ["Reviews", user.reviews],
    ["Bookings", user.reservations],
  ] as const;
  return (
    <>
      <SiteHeader account={account} current="/admin" />
      <Page>
        <Link href="/admin/users" className="mb-4 inline-block text-sm text-muted hover:text-accent">
          ← All users
        </Link>
        <PageTitle
          title={user.display_name}
          subtitle={`${user.email} · joined ${fmt.day(user.created_at, tz)}`}
          action={
            <div className="flex flex-wrap gap-2">
              <ButtonLink variant="secondary" href={`/users/${user.id}`}>
                Profile
              </ButtonLink>
              {canImpersonate && (
                <Button
                  variant="secondary"
                  disabled={impersonating}
                  onClick={async () => {
                    setImpersonating(true);
                    const res = await onImpersonate();
                    setImpersonateError(res.error);
                    if (res.error) setImpersonating(false);
                  }}
                >
                  <VenetianMask className="size-4" /> {impersonating ? "Switching…" : "Log in as this user"}
                </Button>
              )}
              {!user.is_admin && user.id !== account.id && (
                <Button
                  variant={user.banned_at ? "secondary" : "danger"}
                  onClick={() => ban.open({ kind: "user", id: user.id, name: user.display_name, banned: !!user.banned_at })}
                >
                  {user.banned_at ? "Unban user" : "Ban user"}
                </Button>
              )}
            </div>
          }
        />
        {impersonateError && (
          <div className="-mt-4 mb-6">
            <Notice tone="danger">{impersonateError}</Notice>
          </div>
        )}
        <div className="mb-10 flex flex-wrap items-center gap-6">
          <StatusBadge bannedAt={user.banned_at} admin={user.is_admin} />
          {stats.map(([label, n]) => (
            <span key={label} className="text-sm">
              <span className="font-semibold tabular-nums">{n}</span> <span className="text-muted">{label.toLowerCase()}</span>
            </span>
          ))}
        </div>
        {user.banned_at && (
          <div className="mb-10">
            <Notice tone="danger">
              Banned {fmt.day(user.banned_at, tz)}
              {user.ban_reason ? `: ${user.ban_reason}` : "."} Their restaurants and reviews are hidden.
            </Notice>
          </div>
        )}
        {onRename && (
          <div className="mb-10">
            <NameForm key={user.display_name} name={user.display_name} onSave={onRename} />
          </div>
        )}
        <h2 className="mb-4 font-semibold">Restaurants they own</h2>
        {user.owned_restaurants.length === 0 ? (
          <EmptyState title="No restaurants" />
        ) : (
          <RestaurantTable restaurants={user.owned_restaurants} onBan={ban.open} showOwner={false} />
        )}
        {ban.dialog}
      </Page>
    </>
  );
}
