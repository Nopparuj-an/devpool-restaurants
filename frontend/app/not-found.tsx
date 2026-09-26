import { ButtonLink } from "@/components/ui/button";

export default function NotFound() {
  return (
    <main className="mx-auto flex w-full max-w-md flex-1 flex-col items-center justify-center gap-3 px-4 py-24 text-center">
      <p className="text-sm font-medium text-accent">404</p>
      <h1 className="text-2xl font-semibold tracking-tight">We couldn&apos;t find that page</h1>
      <p className="text-sm text-muted">It may have been removed, or the link is wrong.</p>
      <ButtonLink href="/" className="mt-4">
        Back to restaurants
      </ButtonLink>
    </main>
  );
}
