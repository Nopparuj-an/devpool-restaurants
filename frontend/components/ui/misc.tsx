import { Star } from "lucide-react";
import type { ReactNode } from "react";

import { rating as fmtRating, reviewCount } from "@/lib/format";

type Tone = "neutral" | "accent" | "success" | "warning" | "danger";

const tones: Record<Tone, string> = {
  neutral: "bg-surface text-muted",
  accent: "bg-accent-soft text-accent",
  success: "bg-success-soft text-success",
  warning: "bg-warning-soft text-warning",
  danger: "bg-danger-soft text-danger",
};

export function Badge({ tone = "neutral", children }: { tone?: Tone; children: ReactNode }) {
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${tones[tone]}`}>
      {children}
    </span>
  );
}

// "★ 4.7 · 126 reviews", or "No reviews yet" (R-REVIEW-5).
export function Rating({ value, count, size = "sm" }: { value: number | null; count: number; size?: "sm" | "lg" }) {
  if (value === null) return <span className="text-sm text-muted">No reviews yet</span>;
  const big = size === "lg";
  return (
    <span className={`inline-flex items-center gap-1.5 ${big ? "text-base" : "text-sm"}`}>
      <Star className={`${big ? "size-4" : "size-3.5"} fill-accent text-accent`} aria-hidden />
      <span className="font-medium">{fmtRating(value)}</span>
      <span className="text-muted">· {reviewCount(count)}</span>
    </span>
  );
}

// Five static stars for a single review.
export function Stars({ value }: { value: number }) {
  return (
    <span className="inline-flex gap-0.5" aria-label={`${value} out of 5 stars`}>
      {[1, 2, 3, 4, 5].map((n) => (
        <Star key={n} className={`size-3.5 ${n <= value ? "fill-accent text-accent" : "fill-line text-line"}`} aria-hidden />
      ))}
    </span>
  );
}

export function EmptyState({ title, children, action }: { title: string; children?: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 rounded-xl border border-dashed border-line px-6 py-12 text-center">
      <p className="font-medium">{title}</p>
      {children && <p className="max-w-sm text-sm text-muted">{children}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}

export function Notice({ tone = "neutral", children }: { tone?: Tone; children: ReactNode }) {
  return <div className={`rounded-lg px-3 py-2 text-sm ${tones[tone]}`}>{children}</div>;
}

// Cover photo with a quiet fallback when a restaurant has no image URL.
export function Photo({ src, alt = "", className = "" }: { src: string; alt?: string; className?: string }) {
  if (!src) return <div className={`bg-surface ${className}`} />;
  // Plain <img>: images come from our own storage and are already sized.
  // eslint-disable-next-line @next/next/no-img-element
  return <img src={src} alt={alt} className={`bg-surface object-cover ${className}`} />;
}
