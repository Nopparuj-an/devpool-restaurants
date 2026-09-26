"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";

import type { Account } from "@/lib/types";

const nav = [
  { href: "/", label: "Restaurants" },
  { href: "/me/reservations", label: "My bookings" },
  { href: "/me/restaurants", label: "My restaurants" },
];

export function SiteHeader({ account, current }: { account: Account | null; current?: string }) {
  const router = useRouter();
  async function logout() {
    await fetch("/api/auth/logout", { method: "POST" }).catch(() => {});
    router.push("/");
    router.refresh(); // re-render server components logged out
  }
  return (
    <header className="border-b border-line bg-white">
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2 font-semibold tracking-tight">
          <span className="size-2.5 rounded-full bg-accent" aria-hidden />
          Restaurants
        </Link>
        <nav className="hidden gap-1 sm:flex">
          {nav.map((n) => (
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
                <button
                  type="button"
                  onClick={logout}
                  className="w-full rounded-lg px-3 py-2 text-left text-sm hover:bg-surface"
                >
                  Log out
                </button>
              </div>
            </details>
          ) : (
            <Link href="/login" className="font-medium text-accent hover:text-accent-hover">
              Log in
            </Link>
          )}
        </div>
      </div>
      {/* Phone: nav moves to a second, scrollable row. */}
      <nav className="flex gap-1 overflow-x-auto border-t border-line px-2 sm:hidden">
        {nav.map((n) => (
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
