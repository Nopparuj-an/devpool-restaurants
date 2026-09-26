"use client";

import Link from "next/link";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";

import type { Account } from "@/lib/types";

const nav = [
  { href: "/", label: "Restaurants" },
  { href: "/me/reservations", label: "My bookings" },
  { href: "/me/restaurants", label: "My restaurants" },
];

// account: undefined while it loads (the right side stays empty), null when logged out.
export function SiteHeader({ account, current }: { account?: Account | null; current?: string }) {
  const router = useRouter();
  const client = useQueryClient();
  const links = account?.is_admin ? [...nav, { href: "/admin", label: "Admin" }] : nav;
  // Back to the admin's own session (R-ADMIN-7).
  async function stopImpersonating() {
    const res = await fetch("/api/auth/impersonate/stop", { method: "POST" }).catch(() => null);
    client.resetQueries();
    router.push(res?.ok && account ? `/admin/users/${account.id}` : "/");
  }
  async function logout() {
    await fetch("/api/auth/logout", { method: "POST" }).catch(() => {});
    // Forget everything cached for this account; what's on screen refetches logged out.
    client.resetQueries();
    router.push("/");
  }
  return (
    <>
    {account?.impersonator && (
      <div className="bg-ink text-white">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center gap-x-4 gap-y-1 px-4 py-2 text-sm sm:px-6">
          <span>
            You&apos;re using the app as <span className="font-semibold">{account.display_name}</span>.
          </span>
          <button type="button" onClick={stopImpersonating} className="ml-auto font-medium underline underline-offset-4 hover:no-underline">
            Back to {account.impersonator.display_name}
          </button>
        </div>
      </div>
    )}
    <header className="border-b border-line bg-white">
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2 font-semibold tracking-tight">
          <span className="size-2.5 rounded-full bg-accent" aria-hidden />
          Restaurants
        </Link>
        <nav className="hidden gap-1 sm:flex">
          {links.map((n) => (
            <Link
              key={n.href}
              href={n.href}
              aria-current={current === n.href ? "page" : undefined}
              className="rounded-md px-3 py-1.5 text-sm text-muted transition-colors hover:text-ink aria-[current=page]:text-ink aria-[current=page]:font-medium"
            >
              {n.label}
            </Link>
          ))}
        </nav>
        <div className="ml-auto text-sm">
          {account ? (
            <details className="group relative">
              <summary className="flex cursor-pointer list-none items-center gap-2 rounded-md px-1 py-1 hover:bg-surface [&::-webkit-details-marker]:hidden">
                <span className="flex size-7 items-center justify-center rounded-full bg-accent-soft text-xs font-semibold text-accent">
                  {account.display_name.slice(0, 1).toUpperCase()}
                </span>
                <span className="hidden sm:inline">{account.display_name}</span>
              </summary>
              <div className="absolute right-0 z-10 mt-2 w-56 rounded-xl border border-line bg-white p-1 shadow-sm">
                <p className="truncate px-3 py-2 text-xs text-muted">{account.email}</p>
                <Link href={`/users/${account.id}`} className="block rounded-lg px-3 py-2 text-sm hover:bg-surface">
                  Public profile
                </Link>
                <Link href="/me/account" className="block rounded-lg px-3 py-2 text-sm hover:bg-surface">
                  Account settings
                </Link>
                <button
                  type="button"
                  onClick={logout}
                  className="w-full rounded-lg px-3 py-2 text-left text-sm hover:bg-surface"
                >
                  Log out
                </button>
              </div>
            </details>
          ) : account === null ? (
            <Link href="/login" className="font-medium text-accent hover:text-accent-hover">
              Log in
            </Link>
          ) : null}
        </div>
      </div>
      {/* Phone: nav moves to a second, scrollable row. */}
      <nav className="flex gap-1 overflow-x-auto border-t border-line px-2 sm:hidden">
        {links.map((n) => (
          <Link
            key={n.href}
            href={n.href}
            aria-current={current === n.href ? "page" : undefined}
            className="shrink-0 border-b-2 border-transparent px-3 py-2.5 text-sm text-muted aria-[current=page]:border-accent aria-[current=page]:text-ink"
          >
            {n.label}
          </Link>
        ))}
      </nav>
    </header>
    </>
  );
}

export function Page({ children, narrow }: { children: React.ReactNode; narrow?: boolean }) {
  return (
    <main className={`mx-auto w-full flex-1 px-4 py-8 sm:px-6 sm:py-10 ${narrow ? "max-w-md" : "max-w-5xl"}`}>
      {children}
    </main>
  );
}

export function PageTitle({ title, subtitle, action }: { title: string; subtitle?: string; action?: React.ReactNode }) {
  return (
    <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        {subtitle && <p className="mt-1 text-sm text-muted">{subtitle}</p>}
      </div>
      {action}
    </div>
  );
}
