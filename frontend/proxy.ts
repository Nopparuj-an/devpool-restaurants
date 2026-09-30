import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Same-origin API and images so the session cookie just works
// (docs/architecture.md). Read per request, not at build, so one published
// image works everywhere: set API_URL / IMAGES_URL in the container's
// environment (ADR-0016). The defaults suit `pnpm dev` on the host with the
// compose infra running; the Docker image overrides API_URL to http://api:8080.
const apiUrl = () => process.env.API_URL ?? "http://localhost:8080";
// Garage's web endpoint picks the bucket from the Host header (ADR-0005).
const imagesUrl = () =>
  process.env.IMAGES_URL ?? "http://restaurant-images.web.garage.localhost:3902";

export function proxy(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  if (pathname.startsWith("/images/")) {
    return NextResponse.rewrite(
      new URL(pathname.slice("/images".length) + search, imagesUrl()),
    );
  }
  return NextResponse.rewrite(new URL(pathname + search, apiUrl()));
}

export const config = {
  matcher: ["/api/:path*", "/images/:path*"],
};
