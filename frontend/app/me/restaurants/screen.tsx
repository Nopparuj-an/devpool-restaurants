"use client";

import { Plus } from "lucide-react";
import Link from "next/link";

import { Page, PageTitle, SiteHeader } from "@/components/app/site-header";
import { ButtonLink } from "@/components/ui/button";
import { Badge, EmptyState, Photo, Rating } from "@/components/ui/misc";
import type { Account, RestaurantSummary } from "@/lib/types";

export function MyRestaurantsScreen({ account, restaurants }: { account: Account; restaurants: RestaurantSummary[] }) {
  return (
    <>
      <SiteHeader account={account} current="/me/restaurants" />
      <Page>
        <PageTitle
          title="My restaurants"
          action={
            restaurants.length > 0 && (
              <ButtonLink href="/me/restaurants/new">
                <Plus className="size-4" /> Add restaurant
              </ButtonLink>
            )
          }
        />
        {restaurants.length === 0 ? (
          <EmptyState title="You don't have a restaurant yet" action={<ButtonLink href="/me/restaurants/new">Add restaurant</ButtonLink>}>
            Add one and customers can start booking it right away.
          </EmptyState>
        ) : (
          <div className="flex flex-col divide-y divide-line rounded-xl border border-line">
            {restaurants.map((r) => (
              <div key={r.id} className="flex flex-wrap items-center gap-4 p-4">
                <Photo src={r.cover_url} className="size-16 shrink-0 rounded-lg" />
                <div className="flex min-w-0 flex-1 flex-col gap-1">
                  <span className="flex items-center gap-2">
                    <Link href={`/restaurants/${r.id}`} className="truncate font-medium hover:text-accent">
                      {r.name}
                    </Link>
                    {r.banned && <Badge tone="danger">Hidden by an admin</Badge>}
                  </span>
                  <Rating value={r.rating} count={r.review_count} />
                </div>
                <div className="flex gap-2">
                  <ButtonLink size="sm" variant="secondary" href={`/me/restaurants/${r.id}/bookings`}>
                    Bookings
                  </ButtonLink>
                  <ButtonLink size="sm" variant="ghost" href={`/me/restaurants/${r.id}/edit`}>
                    Edit
                  </ButtonLink>
                </div>
              </div>
            ))}
          </div>
        )}
      </Page>
    </>
  );
}
