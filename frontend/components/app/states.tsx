"use client";

// What a route shows before its data is ready: loading, failed, or missing.
import { LoaderCircle } from "lucide-react";

import { Page, SiteHeader } from "@/components/app/site-header";
import { Button, ButtonLink } from "@/components/ui/button";
import type { Account } from "@/lib/types";

export function PageLoading({ account }: { account?: Account | null }) {
  return (
    <>
      <SiteHeader account={account} />
      <Page>
        <div className="flex justify-center py-24" role="status" aria-label="Loading">
          <LoaderCircle className="size-6 animate-spin text-faint" />
        </div>
      </Page>
    </>
  );
}

export function PageError({ account, onRetry }: { account?: Account | null; onRetry: () => void }) {
  return (
    <>
      <SiteHeader account={account} />
      <main className="mx-auto flex w-full max-w-md flex-1 flex-col items-center justify-center gap-3 px-4 py-24 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Something went wrong</h1>
        <p className="text-sm text-muted">We couldn&apos;t load this page. If it keeps happening, the server may be down.</p>
        <Button className="mt-4" onClick={onRetry}>
          Try again
        </Button>
      </main>
    </>
  );
}

export function NotFoundView({ account }: { account?: Account | null }) {
  return (
    <>
      {account !== undefined && <SiteHeader account={account} />}
      <main className="mx-auto flex w-full max-w-md flex-1 flex-col items-center justify-center gap-3 px-4 py-24 text-center">
        <p className="text-sm font-medium text-accent">404</p>
        <h1 className="text-2xl font-semibold tracking-tight">We couldn&apos;t find that page</h1>
        <p className="text-sm text-muted">It may have been removed, or the link is wrong.</p>
        <ButtonLink href="/" className="mt-4">
          Back to restaurants
        </ButtonLink>
      </main>
    </>
  );
}
