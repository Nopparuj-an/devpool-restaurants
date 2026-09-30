"use client";

import Link from "next/link";
import { useState } from "react";

import { Button, buttonClass } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/field";
import { Notice } from "@/components/ui/misc";

export type AuthInput = { email: string; password: string; display_name?: string };

const googleErrors: Record<string, string> = {
  google_state: "That sign-in link expired. Try again.",
  google_denied: "Google sign-in was cancelled.",
  google_email_unverified: "Your Google account's email isn't verified yet.",
  google_conflict: "This email is linked to a different Google account.",
  account_banned: "This account has been suspended.",
};

export function AuthForm({
  mode,
  googleEnabled,
  error: initialError,
  next = "/",
  onSubmit,
}: {
  mode: "login" | "signup";
  googleEnabled: boolean;
  error?: string;
  next?: string;
  onSubmit: (input: AuthInput) => Promise<{ error?: string }>;
}) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState(initialError ? (googleErrors[initialError] ?? "Sign-in failed. Try again.") : "");
  const [busy, setBusy] = useState(false);
  const signup = mode === "signup";

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    const res = await onSubmit({ email, password, ...(signup ? { display_name: name } : {}) });
    setBusy(false);
    setError(res.error ?? "");
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{signup ? "Create an account" : "Log in"}</h1>
        <p className="mt-1 text-sm text-muted">{signup ? "One account books tables and runs your own restaurants." : "Welcome back."}</p>
      </div>

      {googleEnabled && (
        <>
          <a href={`/api/auth/google/start?next=${encodeURIComponent(next)}`} className={buttonClass({ variant: "secondary" }, "w-full")}>
            <GoogleMark />
            Continue with Google
          </a>
          <div className="flex items-center gap-3 text-xs text-faint">
            <span className="h-px flex-1 bg-line" />
            or
            <span className="h-px flex-1 bg-line" />
          </div>
        </>
      )}

      <form onSubmit={submit} className="flex flex-col gap-4">
        {signup && (
          <Field label="Name" hint="Shown on your reviews.">
            <Input value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" required maxLength={80} />
          </Field>
        )}
        <Field label="Email">
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" required />
        </Field>
        <Field label="Password" hint={signup ? "At least 8 characters." : undefined}>
          <Input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete={signup ? "new-password" : "current-password"}
            minLength={signup ? 8 : undefined}
            required
          />
        </Field>
        {error && <Notice tone="danger">{error}</Notice>}
        <Button type="submit" disabled={busy}>
          {busy ? "One moment…" : signup ? "Create account" : "Log in"}
        </Button>
      </form>

      <p className="text-center text-sm text-muted">
        {signup ? "Already have an account? " : "New here? "}
        <Link
          href={`${signup ? "/login" : "/signup"}${next !== "/" ? `?next=${encodeURIComponent(next)}` : ""}`}
          className="font-medium text-accent hover:text-accent-hover"
        >
          {signup ? "Log in" : "Create an account"}
        </Link>
      </p>
    </div>
  );
}

function GoogleMark() {
  return (
    <svg viewBox="0 0 48 48" className="size-4" aria-hidden>
      <path
        fill="#FFC107"
        d="M43.6 20.5H42V20H24v8h11.3C33.7 32.7 29.2 36 24 36c-6.6 0-12-5.4-12-12s5.4-12 12-12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.3-.1-2.4-.4-3.5z"
      />
      <path
        fill="#FF3D00"
        d="m6.3 14.7 6.6 4.8C14.7 15.1 19 12 24 12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 16.3 4 9.7 8.3 6.3 14.7z"
      />
      <path
        fill="#4CAF50"
        d="M24 44c5.2 0 9.9-2 13.4-5.2l-6.2-5.2C29.2 35.1 26.7 36 24 36c-5.2 0-9.6-3.3-11.3-8l-6.5 5C9.5 39.6 16.2 44 24 44z"
      />
      <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.8 2.2-2.2 4.2-4.1 5.6l6.2 5.2C37 39.2 44 34 44 24c0-1.3-.1-2.4-.4-3.5z" />
    </svg>
  );
}
