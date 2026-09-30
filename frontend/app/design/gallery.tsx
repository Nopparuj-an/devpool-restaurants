"use client";

import { useCallback, useState, type ReactNode } from "react";

import { AuthForm } from "@/components/app/auth-form";
import { BookingPanel } from "@/components/app/booking-panel";
import { LoadStrip, OwnerTable } from "@/components/app/owner-table";
import { ReservationCard } from "@/components/app/reservation-card";
import { RestaurantCard } from "@/components/app/restaurant-card";
import { HoursList } from "@/components/app/restaurant-info";
import { ReviewForm, ReviewItem } from "@/components/app/reviews";
import { Button } from "@/components/ui/button";
import { ConfirmDialog, RatingInput, Segmented, Stepper } from "@/components/ui/controls";
import { Field, Input, Select, Textarea } from "@/components/ui/field";
import { Badge, EmptyState, Notice, Rating, Stars } from "@/components/ui/misc";
import * as mock from "@/lib/mock";

import { DAYS } from "./registry";

const COLORS = [
  { name: "accent", hex: "#A80C8A", note: "PEA purple. Buttons, links, selection" },
  { name: "accent-hover", hex: "#8C0A73", note: "Hover and pressed" },
  { name: "accent-soft", hex: "#FBF0F9", note: "Selected backgrounds" },
  { name: "ink", hex: "#0A0A0A", note: "Text" },
  { name: "muted", hex: "#666666", note: "Secondary text" },
  { name: "faint", hex: "#9A9A9A", note: "Placeholders, disabled" },
  { name: "line", hex: "#E8E8E8", note: "Borders, dividers" },
  { name: "surface", hex: "#F7F7F7", note: "Quiet fills" },
];

const STATUS = [
  { name: "success", hex: "#067647" },
  { name: "warning", hex: "#B54708" },
  { name: "danger", hex: "#B42318" },
];

export function Section({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <section id={id} className="scroll-mt-6 border-t border-line py-12">
      <h2 className="mb-8 text-xl font-semibold tracking-tight">{title}</h2>
      {children}
    </section>
  );
}

function Specimen({ id, label, children, wide }: { id: string; label: string; children: ReactNode; wide?: boolean }) {
  return (
    <div id={id} className={`scroll-mt-6 flex flex-col gap-3 ${wide ? "md:col-span-2" : ""}`}>
      <p className="text-xs font-medium uppercase tracking-wide text-faint">{label}</p>
      <div className="flex flex-wrap items-start gap-3">{children}</div>
    </div>
  );
}

export function Foundations() {
  return (
    <Section id="foundations" title="Foundations">
      <div className="grid gap-10">
        <Specimen id="colors" label="Colors">
          <div className="grid w-full grid-cols-2 gap-3 sm:grid-cols-4">
            {COLORS.map((c) => (
              <div key={c.name} className="flex flex-col gap-2">
                <div className="h-16 rounded-lg border border-line" style={{ background: c.hex }} />
                <div className="text-sm">
                  <p className="font-medium">{c.name}</p>
                  <p className="font-mono text-xs text-muted">{c.hex}</p>
                  <p className="text-xs text-muted">{c.note}</p>
                </div>
              </div>
            ))}
          </div>
        </Specimen>
        <Specimen id="status" label="Status">
          {STATUS.map((c) => (
            <div key={c.name} className="flex items-center gap-2 text-sm">
              <span className="size-4 rounded-full" style={{ background: c.hex }} />
              {c.name}
            </div>
          ))}
        </Specimen>
        <Specimen id="type" label="Type, Anuphan">
          <div className="flex flex-col gap-3">
            <p className="text-3xl font-semibold tracking-tight">ครัวริมคลอง, 30 / semibold</p>
            <p className="text-2xl font-semibold tracking-tight">Page title, 24 / semibold</p>
            <p className="text-xl font-semibold tracking-tight">Section title, 20 / semibold</p>
            <p className="font-medium">Card title, 16 / medium</p>
            <p>Body text, 16 / regular. Thai reads the same: จองโต๊ะล่วงหน้าได้</p>
            <p className="text-sm text-muted">Secondary, 14 / muted</p>
            <p className="text-xs text-faint">Caption, 12 / faint</p>
          </div>
        </Specimen>
      </div>
    </Section>
  );
}

export function Components() {
  const [seg, setSeg] = useState("top_rated");
  const [pax, setPax] = useState(2);
  const [stars, setStars] = useState(4);
  const [dialog, setDialog] = useState(false);
  const loadAvailability = useCallback(async (day: string) => mock.availability(Number(day)), []);
  const ok = async () => ({});

  return (
    <Section id="components" title="Components">
      <div className="grid gap-12 md:grid-cols-2">
        <Specimen id="buttons" label="Buttons">
          <Button>Book table</Button>
          <Button variant="secondary">Change</Button>
          <Button variant="ghost">Cancel</Button>
          <Button variant="danger">Delete restaurant</Button>
          <Button size="sm">Small</Button>
          <Button disabled>Disabled</Button>
        </Specimen>

        <Specimen id="badges" label="Badges">
          <Badge tone="accent">Upcoming</Badge>
          <Badge tone="warning">Limited seats left</Badge>
          <Badge tone="success">Now</Badge>
          <Badge>Cancelled</Badge>
          <Badge tone="danger">Error</Badge>
        </Specimen>

        <Specimen id="fields" label="Fields">
          <div className="grid w-full gap-4">
            <Field label="Email">
              <Input placeholder="you@example.com" />
            </Field>
            <Field label="Password" error="Wrong email or password.">
              <Input type="password" aria-invalid defaultValue="secret" />
            </Field>
            <Field label="Cancel until" hint="Before the booking starts">
              <Select defaultValue="30">
                <option value="30">30 min</option>
                <option value="60">1 h</option>
              </Select>
            </Field>
            <Field label="Review">
              <Textarea placeholder="What did you eat? How was it?" />
            </Field>
          </div>
        </Specimen>

        <div className="flex flex-col gap-10">
          <Specimen id="segmented" label="Segmented">
            <Segmented
              label="Sort"
              value={seg}
              onChange={setSeg}
              options={[
                { value: "top_rated", label: "Top rated" },
                { value: "most_reviewed", label: "Most reviewed" },
                { value: "newest", label: "New" },
              ]}
            />
          </Specimen>
          <Specimen id="stepper" label="Stepper">
            <Stepper value={pax} onChange={setPax} min={1} max={10} label="Guests" />
          </Specimen>
          <Specimen id="rating" label="Rating">
            <Rating value={4.7} count={126} />
            <Rating value={4.7} count={126} size="lg" />
            <Rating value={null} count={0} />
            <Stars value={4} />
            <RatingInput value={stars} onChange={setStars} />
          </Specimen>
          <Specimen id="notices" label="Notices">
            <div className="flex w-full flex-col gap-2">
              <Notice>You can&apos;t review your own restaurant.</Notice>
              <Notice tone="success">You&apos;re booked. See it under My bookings.</Notice>
              <Notice tone="danger">Only 2 seats left in that time range.</Notice>
            </div>
          </Specimen>
          <Specimen id="dialog" label="Dialog">
            <Button variant="secondary" onClick={() => setDialog(true)}>
              Open confirm dialog
            </Button>
            <ConfirmDialog
              open={dialog}
              title="Cancel this booking?"
              confirmLabel="Cancel booking"
              danger
              onClose={() => setDialog(false)}
              onConfirm={() => setDialog(false)}
            >
              ครัวริมคลอง, Fri 2 Oct at 12:00 for 4 guests. The seats go back to the restaurant.
            </ConfirmDialog>
          </Specimen>
        </div>

        <Specimen id="empty-state" label="Empty state" wide>
          <div className="w-full">
            <EmptyState title="No upcoming bookings" action={<Button>Find a table</Button>} />
          </div>
        </Specimen>

        <Specimen id="restaurant-card" label="Restaurant card">
          <div className="w-72">
            <RestaurantCard restaurant={mock.summaries[0]} limited />
          </div>
          <div className="w-72">
            <RestaurantCard restaurant={mock.summaries[4]} />
          </div>
        </Specimen>

        <Specimen id="opening-hours" label="Opening hours">
          <HoursList hours={mock.restaurants[0].hours} />
          <HoursList hours={mock.restaurants[3].hours} />
        </Specimen>

        <Specimen id="reservation-card" label="Reservation card" wide>
          <div className="grid w-full gap-3 md:grid-cols-2">
            {mock.myReservations.map((r) => (
              <ReservationCard key={r.id} reservation={r} onCancel={() => {}} onChange={() => {}} />
            ))}
          </div>
        </Specimen>

        <Specimen id="booking-panel" label="Booking panel">
          <div className="w-full max-w-sm">
            <BookingPanel
              seats={10}
              maxMinutes={240}
              cutoffMinutes={30}
              days={DAYS.slice(1)}
              now={mock.MOCK_NOW}
              loadAvailability={loadAvailability}
              onSubmit={ok}
            />
          </div>
        </Specimen>

        <div className="flex flex-col gap-10">
          <Specimen id="review" label="Review">
            <div className="w-full">
              {mock.reviews.slice(0, 2).map((r) => (
                <ReviewItem key={r.id} review={r} />
              ))}
              <ReviewItem review={{ ...mock.reviews[2], author: { ...mock.reviews[2].author, email: "carol@example.com" } }} />
            </div>
          </Specimen>
          <Specimen id="review-form" label="Review form">
            <div className="w-full">
              <ReviewForm mode="customer" onSave={ok} onDelete={async () => {}} />
            </div>
          </Specimen>
          <Specimen id="login" label="Login">
            <div className="w-full max-w-sm rounded-xl border border-line p-6">
              <AuthForm mode="login" googleEnabled onSubmit={ok} />
            </div>
          </Specimen>
        </div>

        <Specimen id="owner-tools" label="Owner: load chart and bookings table" wide>
          <div className="flex w-full flex-col gap-6">
            <LoadStrip reservations={mock.ownerReservations} seats={10} from={mock.bkk(1, "11:00")} to={mock.bkk(1, "22:00")} />
            <OwnerTable reservations={mock.ownerReservations} />
          </div>
        </Specimen>
      </div>
    </Section>
  );
}
