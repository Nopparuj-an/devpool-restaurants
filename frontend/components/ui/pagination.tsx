import { ChevronLeft, ChevronRight } from "lucide-react";
import Link from "next/link";

import { buttonClass } from "./button";

// Previous / next links with "Page 2 of 17". Links (not buttons) so pages are
// shareable and the back button works.
export function Pagination({
  page,
  pageCount,
  hrefFor,
}: {
  page: number;
  pageCount: number;
  hrefFor: (page: number) => string;
}) {
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

// "Show 20 more" for lists that grow in place (reviews, bookings).
export function ShowMore({ shown, total, onMore, busy }: { shown: number; total: number; onMore: () => void; busy?: boolean }) {
  if (shown >= total) return null;
  return (
    <div className="mt-4 flex justify-center">
      <button
        type="button"
        onClick={onMore}
        disabled={busy}
        className={buttonClass({ variant: "secondary", size: "sm" })}
      >
        {busy ? "Loading…" : `Show more (${total - shown} left)`}
      </button>
    </div>
  );
}
