import type { Metadata } from "next";

import { Frame } from "./frame";
import { DesignNav } from "./nav";
import { Components, Foundations, Section } from "./gallery";
import { SCREENS } from "./screen-list";

export const metadata: Metadata = { title: "Design · Restaurants" };

const PHONE = { width: 390, height: 844 };
const DESKTOP = { width: 1280, height: 800 };

// Design system and screens, rendered from the real components with mock
// data (ADR-0008). Each screen is also at /design/screens/<name>.
export default function DesignPage() {
  return (
    <div className="flex min-h-screen flex-col lg:flex-row">
      <DesignNav />
      <main className="mx-auto w-full min-w-0 max-w-6xl flex-1 px-4 py-10 sm:px-6 lg:px-10">
        <header className="flex flex-col gap-2 pb-10">
          <h1 className="text-3xl font-semibold tracking-tight">Design</h1>
          <p className="max-w-prose text-muted">
            The components and screens here are the ones the app uses, filled with sample data. Everything is clickable.
          </p>
        </header>

        <Foundations />
        <Components />

        <Section id="screens" title="Screens">
          <div className="flex flex-col gap-16">
            {SCREENS.map((s) => (
              <div key={s.name} id={`screen-${s.name}`} className="flex scroll-mt-6 flex-col gap-4">
                <h3 className="font-medium">{s.title}</h3>
                <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,3fr)]">
                  <Frame src={`/design/screens/${s.name}`} {...PHONE} label="Phone, 390 wide" />
                  <Frame src={`/design/screens/${s.name}`} {...DESKTOP} label="Desktop, 1280 wide" />
                </div>
              </div>
            ))}
          </div>
        </Section>
      </main>
    </div>
  );
}
