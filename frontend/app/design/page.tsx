import type { Metadata } from "next";

import { Frame } from "./frame";
import { Components, Foundations, Section } from "./gallery";
import { SCREENS } from "./screen-list";

export const metadata: Metadata = { title: "Design · Restaurants" };

const PHONE = { width: 390, height: 844 };
const DESKTOP = { width: 1280, height: 800 };

// Design system and screens, rendered from the real components with mock
// data (ADR-0008). Each screen is also at /design/screens/<name>.
export default function DesignPage() {
  return (
    <main className="mx-auto w-full max-w-6xl px-4 py-10 sm:px-6">
      <header className="flex flex-col gap-2 pb-10">
        <p className="flex items-center gap-2 text-sm font-medium text-accent">
          <span className="size-2.5 rounded-full bg-accent" aria-hidden /> Restaurants
        </p>
        <h1 className="text-3xl font-semibold tracking-tight">Design</h1>
        <p className="max-w-prose text-muted">
          The components and screens here are the ones the app uses, filled with sample data. Everything is clickable.
        </p>
        <nav className="mt-4 flex gap-4 text-sm">
          <a href="#foundations" className="text-muted hover:text-accent">Foundations</a>
          <a href="#components" className="text-muted hover:text-accent">Components</a>
          <a href="#screens" className="text-muted hover:text-accent">Screens</a>
        </nav>
      </header>

      <Foundations />
      <Components />

      <Section id="screens" title="Screens">
        <div className="flex flex-col gap-16">
          {SCREENS.map((s) => (
            <div key={s.name} className="flex flex-col gap-4">
              <h3 className="font-medium">{s.title}</h3>
              <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,3fr)]">
                <Frame src={`/design/screens/${s.name}`} {...PHONE} label="Phone, 390 wide" />
                <Frame src={`/design/screens/${s.name}`} {...DESKTOP} label="Desktop, 1280 wide" />
              </div>
            </div>
          ))}
        </div>
      </Section>
    </main>
  );
}
