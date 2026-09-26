import type { Metadata } from "next";
import { Anuphan } from "next/font/google";
import "./globals.css";

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
      <body className="flex min-h-full flex-col">{children}</body>
    </html>
  );
}
