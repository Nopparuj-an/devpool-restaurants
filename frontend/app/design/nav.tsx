"use client";

import { Menu } from "lucide-react";
import { useEffect, useState } from "react";

import { COMPONENTS, FOUNDATIONS, SCREENS } from "./screen-list";

type Item = { id: string; label: string };
const GROUPS: { title: string; items: Item[] }[] = [
  { title: "Foundations", items: FOUNDATIONS.map(([id, label]) => ({ id, label })) },
  { title: "Components", items: COMPONENTS.map(([id, label]) => ({ id, label })) },
  { title: "Screens", items: SCREENS.map((s) => ({ id: `screen-${s.name}`, label: s.title })) },
];

const IDS = GROUPS.flatMap((g) => g.items.map((i) => i.id));

// Highlights the last anchor that has scrolled past the top of the page.
function useActive() {
  const [active, setActive] = useState(IDS[0]);
  useEffect(() => {
    function update() {
      let current = IDS[0];
      for (const id of IDS) {
        const el = document.getElementById(id);
        if (el && el.getBoundingClientRect().top <= 96) current = id;
      }
      setActive(current);
    }
    update();
    window.addEventListener("scroll", update, { passive: true });
    return () => window.removeEventListener("scroll", update);
  }, []);
  return active;
}

function Links({ active, onPick }: { active: string; onPick?: () => void }) {
  return (
    <nav aria-label="Design sections" className="flex flex-col gap-6 text-sm">
      {GROUPS.map((g) => (
        <div key={g.title}>
          <p className="mb-2 px-3 text-xs font-medium uppercase tracking-wide text-faint">{g.title}</p>
          <ul className="flex flex-col">
            {g.items.map((i) => (
              <li key={i.id}>
                <a
                  href={`#${i.id}`}
                  onClick={onPick}
                  aria-current={active === i.id ? "location" : undefined}
                  className="block rounded-md px-3 py-1.5 text-muted transition-colors hover:text-ink aria-[current=location]:bg-accent-soft aria-[current=location]:font-medium aria-[current=location]:text-accent"
                >
                  {i.label}
                </a>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </nav>
  );
}

export function DesignNav() {
  const active = useActive();
  const [open, setOpen] = useState(false);
  const label = GROUPS.flatMap((g) => g.items).find((i) => i.id === active)?.label;

  return (
    <>
      <aside className="sticky top-0 hidden h-screen w-64 shrink-0 flex-col overflow-y-auto border-r border-line bg-white px-3 py-6 lg:flex">
        <p className="mb-6 flex items-center gap-2 px-3 font-semibold tracking-tight">
          <span className="size-2.5 rounded-full bg-accent" aria-hidden /> Design
        </p>
        <Links active={active} />
      </aside>

      <div className="sticky top-0 z-30 border-b border-line bg-white lg:hidden">
        <button
          type="button"
          aria-expanded={open}
          onClick={() => setOpen(!open)}
          className="flex h-12 w-full items-center gap-3 px-4 text-sm"
        >
          <Menu className="size-4" aria-hidden />
          <span className="font-semibold">Design</span>
          <span className="truncate text-muted">{label}</span>
        </button>
        {open && (
          <div className="max-h-[70vh] overflow-y-auto border-t border-line px-2 py-4">
            <Links active={active} onPick={() => setOpen(false)} />
          </div>
        )}
      </div>
    </>
  );
}
