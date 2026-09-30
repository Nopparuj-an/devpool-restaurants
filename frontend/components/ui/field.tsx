import type { ComponentProps, ReactNode } from "react";

const control =
  "w-full rounded-lg border border-line bg-white px-3 text-sm text-ink placeholder:text-faint transition-colors hover:border-ink/30 focus-visible:border-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/20 disabled:bg-surface disabled:text-faint aria-invalid:border-danger";

export function Input({ className = "", ...props }: ComponentProps<"input">) {
  return <input className={`${control} h-10 ${className}`} {...props} />;
}

export function Textarea({ className = "", ...props }: ComponentProps<"textarea">) {
  return <textarea className={`${control} min-h-24 py-2 leading-relaxed ${className}`} {...props} />;
}

// The native arrow is dropped for a chevron drawn as a background, so it sits
// the same in every browser.
const chevron =
  "appearance-none bg-[url(data:image/svg+xml,%3Csvg%20xmlns=%27http://www.w3.org/2000/svg%27%20viewBox=%270%200%2016%2016%27%20fill=%27none%27%20stroke=%27%236b7280%27%20stroke-width=%271.5%27%20stroke-linecap=%27round%27%20stroke-linejoin=%27round%27%3E%3Cpath%20d=%27m4%206%204%204%204-4%27/%3E%3C/svg%3E)] bg-size-[1rem] bg-position-[right_0.75rem_center] bg-no-repeat";

export function Select({ className = "", ...props }: ComponentProps<"select">) {
  return <select className={`${control} ${chevron} h-10 pr-9 ${className}`} {...props} />;
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
      {error ? <span className="text-sm text-danger">{error}</span> : hint ? <span className="text-sm text-muted">{hint}</span> : null}
    </label>
  );
}
