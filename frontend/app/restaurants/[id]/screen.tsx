"use client";

import { Pencil, Table2 } from "lucide-react";
import Link from "next/link";

import type { BanTarget } from "@/components/app/admin";
import { BookingPanel, type BookingInput } from "@/components/app/booking-panel";
import { Gallery, HoursList } from "@/components/app/restaurant-info";
import { ReviewBreakdown, ReviewForm, ReviewItem, ReviewSortControl } from "@/components/app/reviews";
import { Page, SiteHeader } from "@/components/app/site-header";
import { Button, ButtonLink } from "@/components/ui/button";
import { Notice, Rating } from "@/components/ui/misc";
import { LoadMore } from "@/components/ui/pagination";
import * as fmt from "@/lib/format";
import type { Account, ActionResult, Availability, Day, Reservation, RestaurantDetail, Review, ReviewSort } from "@/lib/types";

export function RestaurantScreen({
  account,
  restaurant: r,
  reviews,
  myReview,
  days,
  now,
  loadAvailability,
  onBook,
  onSaveReview,
  onDeleteReview,
  editing,
  initialDay,
  reviewsTotal,
  onMoreReviews,
  reviewCounts,
  reviewRating = 0,
  reviewSort = "newest",
  onReviewFilter,
  onAdminBan,
}: {
  account: Account | null;
  restaurant: RestaurantDetail;
  // Admins get ban / unban here too (R-ADMIN-4).
  onAdminBan?: (target: BanTarget) => void;
  reviews: Review[];
  // Reviews matching the filter; `reviews` may be the first pages only.
  reviewsTotal?: number;
  onMoreReviews?: () => Promise<void>;
  // Per-star counts and the filter / sort they drive (R-REVIEW-8).
  reviewCounts?: Record<string, number>;
  reviewRating?: number;
  reviewSort?: ReviewSort;
  onReviewFilter?: (f: { rating: number; sort: ReviewSort }) => void;
  myReview?: Review;
  days: Day[];
  now: string;
  loadAvailability: (day: string) => Promise<Availability>;
  onBook: (input: BookingInput) => ActionResult;
  onSaveReview: (input: { rating: number; body: string }) => ActionResult;
  onDeleteReview: () => Promise<void>;
  editing?: Reservation;
  initialDay?: string;
}) {
  const others = reviews.filter((rv) => rv.id !== myReview?.id);
  return (
    <>
      <SiteHeader account={account} current="/" />
      <Page>
        <div className="flex flex-col gap-8">
          {(r.banned || r.owner_banned) && (
            <Notice tone="warning">
              <span className="font-medium">Hidden from customers.</span>{" "}
              {r.banned
                ? `An admin banned this restaurant${r.ban_reason ? `: ${r.ban_reason}` : "."}`
                : "The owner's account is suspended."}
            </Notice>
          )}
          {account?.is_admin && onAdminBan && !r.is_owner && (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line px-4 py-3 text-sm">
              <span className="text-muted">Admin</span>
              <span className="flex flex-wrap gap-2">
                <ButtonLink size="sm" variant="secondary" href={`/admin/users/${r.owner.id}`}>
                  Owner
                </ButtonLink>
                {r.can_manage && (
                  <>
                    <ButtonLink size="sm" variant="secondary" href={`/me/restaurants/${r.id}/bookings`}>
                      Bookings
                    </ButtonLink>
                    <ButtonLink size="sm" variant="secondary" href={`/me/restaurants/${r.id}/edit`}>
                      Edit
                    </ButtonLink>
                  </>
                )}
                <Button
                  size="sm"
                  variant={r.banned ? "secondary" : "danger"}
                  onClick={() => onAdminBan({ kind: "restaurant", id: r.id, name: r.name, banned: !!r.banned })}
                >
                  {r.banned ? "Unban restaurant" : "Ban restaurant"}
                </Button>
              </span>
            </div>
          )}
          {r.is_owner && (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-accent-soft px-4 py-3 text-sm">
              <span>
                This is your restaurant. {r.upcoming_reservations ?? 0} upcoming {r.upcoming_reservations === 1 ? "booking" : "bookings"}.
              </span>
              <span className="flex gap-2">
                <ButtonLink size="sm" variant="secondary" href={`/me/restaurants/${r.id}/bookings`}>
                  <Table2 className="size-4" /> Bookings
                </ButtonLink>
                <ButtonLink size="sm" variant="secondary" href={`/me/restaurants/${r.id}/edit`}>
                  <Pencil className="size-4" /> Edit
                </ButtonLink>
              </span>
            </div>
          )}

          <Gallery images={r.images} name={r.name} />

          {/* Phones: intro, booking, details. Desktop: booking is a sticky right column. */}
          <div className="grid grid-cols-[minmax(0,1fr)] gap-10 lg:grid-cols-[minmax(0,1fr)_24rem]">
            <div className="flex flex-col gap-2 lg:col-start-1">
              <h1 className="text-3xl font-semibold tracking-tight">{r.name}</h1>
              <p className="text-muted">
                {r.cuisine} · {r.location}
              </p>
              <p className="text-sm text-muted">
                Run by{" "}
                <Link href={`/users/${r.owner.id}`} className="font-medium text-ink hover:text-accent">
                  {r.owner.display_name}
                </Link>
              </p>
              <Rating value={r.rating} count={r.review_count} size="lg" />
              <p className="mt-3 max-w-prose leading-relaxed">{r.description}</p>
            </div>

            <aside className="min-w-0 lg:sticky lg:top-20 lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:self-start">
              <BookingPanel
                seats={r.seats}
                maxMinutes={r.max_reservation_minutes}
                cutoffMinutes={r.cancel_cutoff_minutes}
                days={days}
                now={now}
                loadAvailability={loadAvailability}
                onSubmit={onBook}
                editing={editing}
                initialDay={initialDay}
                locked={!account}
              />
            </aside>

            <div className="flex min-w-0 flex-col gap-10 lg:col-start-1">
              <section className="flex flex-col gap-4">
                <h2 className="font-semibold">Opening hours</h2>
                <HoursList hours={r.hours} />
                <p className="text-sm text-muted">{r.seats} seats</p>
              </section>

              <section className="flex flex-col gap-4">
                <h2 className="font-semibold">
                  Reviews {r.review_count > 0 && <span className="font-normal text-muted">({r.review_count.toLocaleString("en")})</span>}
                </h2>
                {reviewCounts && onReviewFilter && r.review_count > 0 && (
                  <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                    <ReviewBreakdown
                      counts={reviewCounts}
                      total={r.review_count}
                      rating={reviewRating}
                      onRating={(rating) => onReviewFilter({ rating, sort: reviewSort })}
                    />
                    <ReviewSortControl value={reviewSort} onChange={(sort) => onReviewFilter({ rating: reviewRating, sort })} />
                  </div>
                )}
                <ReviewForm
                  mode={!account ? "anonymous" : r.is_owner ? "owner" : "customer"}
                  existing={myReview}
                  onSave={onSaveReview}
                  onDelete={onDeleteReview}
                />
                {reviewRating > 0 && (
                  <p className="text-sm text-muted">
                    Showing {fmt.count(reviewsTotal ?? others.length, `${reviewRating} star review`)}.{" "}
                    <button
                      type="button"
                      className="text-accent hover:underline"
                      onClick={() => onReviewFilter?.({ rating: 0, sort: reviewSort })}
                    >
                      Show all
                    </button>
                  </p>
                )}
                {others.length === 0 ? (
                  <p className="text-sm text-muted">{reviewRating > 0 ? "No other reviews with that rating." : "No reviews yet."}</p>
                ) : (
                  <div>
                    {others.map((rv) => (
                      <ReviewItem key={rv.id} review={rv} />
                    ))}
                    {onMoreReviews && <LoadMore shown={reviews.length} total={reviewsTotal ?? reviews.length} onMore={onMoreReviews} />}
                  </div>
                )}
              </section>
            </div>
          </div>
        </div>
      </Page>
    </>
  );
}
