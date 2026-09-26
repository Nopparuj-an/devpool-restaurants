"use client";

import { BadgeCheck } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { RatingInput } from "@/components/ui/controls";
import { Textarea } from "@/components/ui/field";
import { Notice, Stars } from "@/components/ui/misc";
import * as fmt from "@/lib/format";
import type { Review } from "@/lib/types";
import { useTimeZone } from "@/lib/use-time-zone";

export function ReviewItem({ review: r }: { review: Review }) {
  const tz = useTimeZone();
  return (
    <article className="flex flex-col gap-2 border-b border-line py-5 last:border-0">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <Link href={`/users/${r.author.id}`} className="font-medium hover:text-accent">
          {r.author.display_name}
        </Link>
        {r.author.email && <span className="text-sm text-muted">{r.author.email}</span>}
        {r.verified && (
          <span className="inline-flex items-center gap-1 text-xs text-success">
            <BadgeCheck className="size-3.5" aria-hidden />
            Visited
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <Stars value={r.rating} />
        <span className="text-xs text-faint">{fmt.day(r.updated_at, tz)}</span>
      </div>
      <p className="text-sm leading-relaxed">{r.body}</p>
    </article>
  );
}

type Mode = "anonymous" | "owner" | "customer";

// Write, edit or delete your one review of a restaurant (R-REVIEW-2, -3, -7).
export function ReviewForm({
  mode,
  existing,
  onSave,
  onDelete,
}: {
  mode: Mode;
  existing?: Review;
  onSave: (input: { rating: number; body: string }) => Promise<{ error?: string }>;
  onDelete: () => Promise<void>;
}) {
  const [rating, setRating] = useState(existing?.rating ?? 0);
  const [body, setBody] = useState(existing?.body ?? "");
  const [editing, setEditing] = useState(!existing);
  const [message, setMessage] = useState<{ tone: "success" | "danger"; text: string } | null>(null);

  if (mode === "owner") return <Notice>You can&apos;t review your own restaurant.</Notice>;
  if (mode === "anonymous") return <Notice>Log in to write a review.</Notice>;

  if (!editing && existing) {
    return (
      <div className="flex flex-col gap-3 rounded-xl border border-line p-5">
        <p className="text-sm font-medium">Your review</p>
        <Stars value={existing.rating} />
        <p className="text-sm leading-relaxed">{existing.body}</p>
        <div className="flex gap-2">
          <Button size="sm" variant="secondary" onClick={() => setEditing(true)}>
            Edit
          </Button>
          <Button size="sm" variant="ghost" onClick={() => onDelete()}>
            Delete
          </Button>
        </div>
      </div>
    );
  }

  async function save() {
    if (rating === 0) return setMessage({ tone: "danger", text: "Pick a star rating." });
    if (!body.trim()) return setMessage({ tone: "danger", text: "Write a few words about your visit." });
    const res = await onSave({ rating, body });
    setMessage(res.error ? { tone: "danger", text: res.error } : { tone: "success", text: "Thanks. Your review is up." });
    if (!res.error) setEditing(false);
  }

  return (
    <div className="flex flex-col gap-4 rounded-xl border border-line p-5">
      <p className="text-sm font-medium">{existing ? "Edit your review" : "Write a review"}</p>
      <RatingInput value={rating} onChange={setRating} />
      <Textarea
        value={body}
        onChange={(e) => setBody(e.target.value)}
        placeholder="What did you eat? How was it?"
        maxLength={2000}
        aria-label="Review"
      />
      <div className="flex gap-2">
        <Button onClick={save}>Post review</Button>
        {existing && (
          <Button variant="ghost" onClick={() => setEditing(false)}>
            Cancel
          </Button>
        )}
      </div>
      {message && <Notice tone={message.tone}>{message.text}</Notice>}
    </div>
  );
}
