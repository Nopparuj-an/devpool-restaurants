"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { buttonClass } from "./button";

// Previous / next links with "Page 2 of 17". Links (not buttons) so pages are
// shareable and the back button works. Admin lists use it; public lists grow
// with LoadMore instead.
export function Pagination({ page, pageCount, hrefFor }: { page: number; pageCount: number; hrefFor: (page: number) => string }) {
  if (pageCount <= 1) return null;
  const link = (to: number, label: string, icon: React.ReactNode, disabled: boolean) =>
    disabled ? (
      <span className={buttonClass({ variant: "secondary", size: "sm" }, "pointer-events-none opacity-40")} aria-disabled>
        {icon}
        {label}
      </span>
    ) : (
      <Link href={hrefFor(to)} className={buttonClass({ variant: "secondary", size: "sm" })} scroll>
        {icon}
        {label}
      </Link>
    );
  return (
    <nav aria-label="Pages" className="mt-12 flex items-center justify-between gap-4">
      {link(page - 1, "Previous", <ChevronLeft className="size-4" />, page <= 1)}
      <span className="text-sm text-muted tabular-nums">
        Page {page} of {pageCount}
      </span>
      {link(page + 1, "Next", <ChevronRight className="size-4 order-last" />, page >= pageCount)}
    </nav>
  );
}

// Endless scroll for lists that grow in place (restaurants, reviews,
// bookings). The next page loads when this button comes near the viewport;
// it stays a real button for keyboards and in case the observer never fires.
// auto={false} for a list with another list below it, which would otherwise
// be pushed out of reach: then it loads on click only.
export function LoadMore({
  shown,
  total,
  onMore,
  auto = true,
}: {
  shown: number;
  total: number;
  onMore: () => Promise<void>;
  auto?: boolean;
}) {
  const ref = useRef<HTMLButtonElement>(null);
  const [busy, setBusy] = useState(false);
  const loading = useRef(false);
  const latest = useRef(onMore);
  useEffect(() => {
    latest.current = onMore;
  });
  const more = shown < total;

  async function load() {
    if (loading.current) return;
    loading.current = true;
    setBusy(true);
    try {
      await latest.current();
    } finally {
      loading.current = false;
      setBusy(false);
    }
  }

  // A new observer after each page reports straight away whether the button
  // is still in view (a short page), so loading goes on until it isn't.
  useEffect(() => {
    const el = ref.current;
    if (!el || !auto || !more || busy) return;
    const io = new IntersectionObserver((entries) => entries.some((e) => e.isIntersecting) && load(), {
      rootMargin: "600px 0px",
    });
    io.observe(el);
    return () => io.disconnect();
  }, [auto, more, busy, shown]);

  if (!more) return null;
  return (
    <div className="mt-6 flex justify-center">
      <button ref={ref} type="button" onClick={load} disabled={busy} className={buttonClass({ variant: "secondary", size: "sm" })}>
        {busy ? "Loading…" : `Show more (${(total - shown).toLocaleString("en")} left)`}
      </button>
    </div>
  );
}
