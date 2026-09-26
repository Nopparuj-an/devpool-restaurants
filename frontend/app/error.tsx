"use client";

import { Button } from "@/components/ui/button";

export default function Error({ reset }: { error: Error; reset: () => void }) {
  return (
    <main className="mx-auto flex w-full max-w-md flex-1 flex-col items-center justify-center gap-3 px-4 py-24 text-center">
      <h1 className="text-2xl font-semibold tracking-tight">Something went wrong</h1>
      <p className="text-sm text-muted">We couldn&apos;t load this page. If it keeps happening, the server may be down.</p>
      <Button className="mt-4" onClick={reset}>
        Try again
      </Button>
    </main>
  );
}
