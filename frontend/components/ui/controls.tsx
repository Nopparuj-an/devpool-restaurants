"use client";

import { Minus, Plus, Star } from "lucide-react";
import { useEffect, useRef, type ReactNode } from "react";

import { Button } from "./button";

// Pill-style single choice (tabs, sort order, filters).
export function Segmented<T extends string>({
  options,
  value,
  onChange,
  label,
}: {
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
  label: string;
}) {
  return (
    <div role="radiogroup" aria-label={label} className="inline-flex rounded-lg bg-surface p-0.5">
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          role="radio"
          aria-checked={o.value === value}
          onClick={() => onChange(o.value)}
          className={`h-8 rounded-md px-3 text-sm transition-colors ${
            o.value === value ? "bg-white font-medium text-ink shadow-sm" : "text-muted hover:text-ink"
          }`}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function Stepper({
  value,
  onChange,
  min,
  max,
  label,
}: {
  value: number;
  onChange: (value: number) => void;
  min: number;
  max: number;
  label: string;
}) {
  const btn =
    "flex size-10 items-center justify-center text-ink transition-colors hover:bg-surface disabled:text-faint disabled:hover:bg-transparent";
  return (
    <div className="inline-flex h-10 items-center rounded-lg border border-line" role="group" aria-label={label}>
      <button type="button" className={`${btn} rounded-l-lg`} onClick={() => onChange(value - 1)} disabled={value <= min} aria-label="Fewer">
        <Minus className="size-4" />
      </button>
      <span className="w-10 text-center text-sm font-medium tabular-nums" aria-live="polite">
        {value}
      </span>
      <button type="button" className={`${btn} rounded-r-lg`} onClick={() => onChange(value + 1)} disabled={value >= max} aria-label="More">
        <Plus className="size-4" />
      </button>
    </div>
  );
}

export function RatingInput({ value, onChange }: { value: number; onChange: (value: number) => void }) {
  return (
    <div role="radiogroup" aria-label="Rating" className="inline-flex gap-1">
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          role="radio"
          aria-checked={n === value}
          aria-label={`${n} star${n === 1 ? "" : "s"}`}
          onClick={() => onChange(n)}
          className="rounded p-0.5"
        >
          <Star className={`size-6 transition-colors ${n <= value ? "fill-accent text-accent" : "fill-white text-line hover:text-accent/50"}`} />
        </button>
      ))}
    </div>
  );
}

// Modal confirmation built on <dialog>.
export function ConfirmDialog({
  open,
  title,
  children,
  confirmLabel,
  danger,
  onConfirm,
  onClose,
}: {
  open: boolean;
  title: string;
  children: ReactNode;
  confirmLabel: string;
  danger?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) d.showModal();
    if (!open && d.open) d.close();
  }, [open]);
  return (
    <dialog
      ref={ref}
      onClose={onClose}
      className="m-auto w-[min(28rem,calc(100%-2rem))] rounded-xl p-0 backdrop:bg-ink/30"
    >
      <div className="flex flex-col gap-3 p-6">
        <h2 className="text-lg font-semibold">{title}</h2>
        <div className="text-sm text-muted">{children}</div>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant={danger ? "danger" : "primary"} onClick={onConfirm}>
            {confirmLabel}
          </Button>
        </div>
      </div>
    </dialog>
  );
}
