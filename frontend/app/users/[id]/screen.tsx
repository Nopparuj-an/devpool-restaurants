"use client";

import { BadgeCheck, EyeOff } from "lucide-react";
import Link from "next/link";

import { RestaurantCard } from "@/components/app/restaurant-card";
import { Page, SiteHeader } from "@/components/app/site-header";
import { ButtonLink } from "@/components/ui/button";
import { Notice, Photo, Stars } from "@/components/ui/misc";
import { LoadMore } from "@/components/ui/pagination";
import * as fmt from "@/lib/format";
import type { Account, Profile, ProfileReview, RestaurantSummary } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

// A public profile (R-PROFILE-1): name, when they joined, their restaurants
// and reviews. Never the email.
export function ProfileScreen({
  account,
  profile: p,
  restaurants,
  restaurantsTotal,
  onMoreRestaurants,
  reviews,
  reviewsTotal,
  onMoreReviews,
}: {
  account: Account | null;
  profile: Profile;
  restaurants: RestaurantSummary[];
  restaurantsTotal: number;
  onMoreRestaurants?: () => Promise<void>;
  reviews: ProfileReview[];
  reviewsTotal: number;
  onMoreReviews?: () => Promise<void>;
}) {
  const tz = useTimeZone();
  const self = account?.id === p.id;
  return (
    <>
      <SiteHeader account={account} />
      <Page>
        {p.banned && (
          <div className="mb-6">
            <Notice tone="danger">This account is banned. Only admins can see this page.</Notice>
          </div>
        )}
        <div className="mb-10 flex flex-wrap items-center gap-5">
          <span className="flex size-16 items-center justify-center rounded-full bg-accent-soft text-2xl font-semibold text-accent">
            {p.display_name.slice(0, 1).toUpperCase()}
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <h1 className="text-2xl font-semibold tracking-tight">{p.display_name}</h1>
            <p className="text-sm text-muted">
              Joined {fmt.day(p.created_at, tz)} · {fmt.count(p.review_count, "review")} · {fmt.count(p.restaurant_count, "restaurant")}
            </p>
          </div>
          <div className="flex gap-2">
            {self && (
              <ButtonLink variant="secondary" href="/me/account">
                Edit profile
              </ButtonLink>
            )}
            {account?.is_admin && !self && (
              <ButtonLink variant="secondary" href={`/admin/users/${p.id}`}>
                Manage
              </ButtonLink>
            )}
          </div>
        </div>

        {restaurants.length > 0 && (
          <section className="mb-12">
            <h2 className="mb-4 font-semibold">Restaurants</h2>
            <div className="grid gap-x-6 gap-y-10 sm:grid-cols-2 lg:grid-cols-3">
              {restaurants.map((r) => (
                <RestaurantCard key={r.id} restaurant={r} />
              ))}
            </div>
            {onMoreRestaurants && (
              // Reviews come below, so this list grows on click only.
              <LoadMore shown={restaurants.length} total={restaurantsTotal} onMore={onMoreRestaurants} auto={false} />
            )}
          </section>
        )}

        <section>
          <h2 className="mb-2 font-semibold">Reviews</h2>
          {reviews.length === 0 ? (
            <p className="text-sm text-muted">No reviews yet.</p>
          ) : (
            <div>
              {reviews.map((rv) => (
                <article key={rv.id} className="flex gap-4 border-b border-line py-5 last:border-0">
                  <Link href={`/restaurants/${rv.restaurant.id}`} className="shrink-0">
                    <Photo src={rv.restaurant.cover_url} className="size-14 rounded-lg" />
                  </Link>
                  <div className="flex min-w-0 flex-col gap-2">
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                      <Link href={`/restaurants/${rv.restaurant.id}`} className="font-medium hover:text-accent">
                        {rv.restaurant.name}
                      </Link>
                      {rv.verified && (
                        <span className="inline-flex items-center gap-1 text-xs text-success">
                          <BadgeCheck className="size-3.5" aria-hidden />
                          Visited
                        </span>
                      )}
                      {rv.hidden && (
                        <span className="inline-flex items-center gap-1 text-xs text-muted">
                          <EyeOff className="size-3.5" aria-hidden />
                          Restaurant hidden
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <Stars value={rv.rating} />
                      <span className="text-xs text-faint">{fmt.day(rv.updated_at, tz)}</span>
                    </div>
                    <p className="text-sm leading-relaxed">{rv.body}</p>
                  </div>
                </article>
              ))}
              {onMoreReviews && <LoadMore shown={reviews.length} total={reviewsTotal} onMore={onMoreReviews} />}
            </div>
          )}
        </section>
      </Page>
    </>
  );
}
