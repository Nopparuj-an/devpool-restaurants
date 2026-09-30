import type { NextConfig } from "next";

// /api and /images are proxied at request time by proxy.ts, so the backend
// URLs are runtime environment variables, not build args (ADR-0016).
const nextConfig: NextConfig = {
  output: "standalone",
  experimental: {
    // proxy.ts buffers request bodies; restaurant creation uploads up to
    // 10 images x 10 MB (backend/internal/restaurant/image_handler.go).
    proxyClientMaxBodySize: "110mb",
  },
};

export default nextConfig;
