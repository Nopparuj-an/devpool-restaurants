import type { NextConfig } from "next";

// Rewrite destinations are serialized at `next build`, so the Docker image
// receives them as build args (see Dockerfile). Defaults suit `pnpm dev` on
// the host with the compose infra running.
const apiUrl = process.env.API_URL ?? "http://localhost:8080";
// Garage's web endpoint picks the bucket from the Host header (ADR-0005).
const imagesUrl =
  process.env.IMAGES_URL ?? "http://restaurant-images.web.garage.localhost:3902";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [
      // Same-origin API so the session cookie just works (docs/architecture.md).
      { source: "/api/:path*", destination: `${apiUrl}/api/:path*` },
      { source: "/images/:path*", destination: `${imagesUrl}/:path*` },
    ];
  },
};

export default nextConfig;
