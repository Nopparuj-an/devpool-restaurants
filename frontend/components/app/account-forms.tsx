"use client";

import { BadgeCheck } from "lucide-react";
import { useState, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/field";
import { Notice } from "@/components/ui/misc";
import type { Account } from "@/lib/types";

type Save = Promise<{ error?: string }>;
type Message = { tone: "success" | "danger"; text: string } | null;

function Section({ title, hint, children }: { title: string; hint: string; children: ReactNode }) {
  return (
    <section className="grid gap-6 border-b border-line py-8 first:pt-0 last:border-0 md:grid-cols-[14rem_1fr]">
      <div>
        <h2 className="font-medium">{title}</h2>
        <p className="mt-1 text-sm text-muted">{hint}</p>
      </div>
      <div className="flex max-w-md flex-col gap-4">{children}</div>
    </section>
  );
}

export function ProfileForm({ account, onSave }: { account: Account; onSave: (name: string) => Save }) {
  const [name, setName] = useState(account.display_name);
  const [message, setMessage] = useState<Message>(null);
  const [busy, setBusy] = useState(false);
  const changed = name.trim() !== account.display_name;

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    const res = await onSave(name);
    setBusy(false);
    setMessage(res.error ? { tone: "danger", text: res.error } : { tone: "success", text: "Name updated." });
  }

  return (
    <Section title="Profile" hint="Your name is shown on your reviews and to restaurants you book.">
      <form onSubmit={save} className="flex flex-col gap-4">
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} required autoComplete="name" />
        </Field>
        <Field
          label="Email"
          hint={
            account.email_verified ? (
              <span className="inline-flex items-center gap-1 text-success">
                <BadgeCheck className="size-3.5" /> Verified with Google
              </span>
            ) : (
              "Used to log in. It can't be changed."
            )
          }
        >
          <Input value={account.email} disabled />
        </Field>
        <div className="flex items-center gap-3">
          <Button type="submit" disabled={!changed || busy}>
            {busy ? "Saving…" : "Save name"}
          </Button>
          {message && <Notice tone={message.tone}>{message.text}</Notice>}
        </div>
      </form>
    </Section>
  );
}

// Change the password, or set one for accounts that only use Google (ADR-0002).
export function PasswordForm({
  hasPassword,
  onSave,
}: {
  hasPassword: boolean;
  onSave: (current: string, next: string) => Save;
}) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [message, setMessage] = useState<Message>(null);
  const [busy, setBusy] = useState(false);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (next !== confirm) return setMessage({ tone: "danger", text: "The new passwords don't match." });
    setBusy(true);
    const res = await onSave(current, next);
    setBusy(false);
    if (res.error) return setMessage({ tone: "danger", text: res.error });
    setCurrent("");
    setNext("");
    setConfirm("");
    setMessage({ tone: "success", text: hasPassword ? "Password changed." : "Password set. You can now log in with email too." });
  }

  return (
    <Section
      title="Password"
      hint={hasPassword ? "Use at least 8 characters." : "You log in with Google. Add a password to also log in with your email."}
    >
      <form onSubmit={save} className="flex flex-col gap-4">
        {hasPassword && (
          <Field label="Current password">
            <Input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" required />
          </Field>
        )}
        <Field label="New password">
          <Input type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" minLength={8} required />
        </Field>
        <Field label="Repeat new password">
          <Input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" minLength={8} required />
        </Field>
        <div className="flex items-center gap-3">
          <Button type="submit" disabled={busy}>
            {busy ? "Saving…" : hasPassword ? "Change password" : "Set password"}
          </Button>
          {message && <Notice tone={message.tone}>{message.text}</Notice>}
        </div>
      </form>
    </Section>
  );
}
