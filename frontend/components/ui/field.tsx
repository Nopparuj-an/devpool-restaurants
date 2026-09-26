import type { ComponentProps, ReactNode } from "react";

const control =
  "w-full rounded-lg border border-line bg-white px-3 text-sm text-ink placeholder:text-faint transition-colors hover:border-ink/30 focus-visible:border-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/20 disabled:bg-surface disabled:text-faint aria-invalid:border-danger";

export function Input({ className = "", ...props }: ComponentProps<"input">) {
  return <input className={`${control} h-10 ${className}`} {...props} />;
}

export function Textarea({ className = "", ...props }: ComponentProps<"textarea">) {
  return <textarea className={`${control} min-h-24 py-2 leading-relaxed ${className}`} {...props} />;
}

export function Select({ className = "", ...props }: ComponentProps<"select">) {
  return <select className={`${control} h-10 pr-8 ${className}`} {...props} />;
}

// A labelled control with optional hint and error text.
export function Field({
  label,
  hint,
  error,
  children,
  className = "",
}: {
  label: string;
  hint?: ReactNode;
  error?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <label className={`flex flex-col gap-1.5 ${className}`}>
      <span className="text-sm font-medium">{label}</span>
      {children}
      {error ? (
        <span className="text-sm text-danger">{error}</span>
      ) : hint ? (
        <span className="text-sm text-muted">{hint}</span>
      ) : null}
    </label>
  );
}
