import type { Metadata } from "next";
import { Anuphan } from "next/font/google";
import { Suspense } from "react";

import { PageLoading } from "@/components/app/states";

import "./globals.css";
import { Providers } from "./providers";

// Anuphan covers Thai and Latin, so Thai restaurant names and English UI
// share one typeface.
const anuphan = Anuphan({
  variable: "--font-anuphan",
  subsets: ["latin", "thai"],
});

export const metadata: Metadata = {
  title: "Restaurants",
  description: "Book a table and review restaurants",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={`${anuphan.variable} h-full`}>
      <body className="flex min-h-full flex-col">
        <Providers>
          {/* Routes read ?query= in the browser, so their HTML is only a shell. */}
          <Suspense fallback={<PageLoading />}>{children}</Suspense>
        </Providers>
      </body>
    </html>
  );
}
