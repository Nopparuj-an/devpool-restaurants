"use client";

import { ExternalLink } from "lucide-react";
import { useEffect, useRef, useState } from "react";

// A live screen in an iframe at a real device width, scaled down to fit.
export function Frame({ src, width, height, label }: { src: string; width: number; height: number; label: string }) {
  const box = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(0.3);

  useEffect(() => {
    const el = box.current;
    if (!el) return;
    const ro = new ResizeObserver(([entry]) => setScale(Math.min(1, entry.contentRect.width / width)));
    ro.observe(el);
    return () => ro.disconnect();
  }, [width]);

  return (
    <figure className="flex flex-col gap-2">
      <div ref={box} className="w-full">
        <div
          className="overflow-hidden rounded-xl border border-line bg-white"
          style={{ width: width * scale, height: height * scale }}
        >
          <iframe
            src={src}
            title={label}
            loading="lazy"
            style={{ width, height, transform: `scale(${scale})`, transformOrigin: "0 0" }}
          />
        </div>
      </div>
      <figcaption className="flex items-center justify-between text-xs text-muted">
        <span>{label}</span>
        <a href={src} target="_blank" className="inline-flex items-center gap-1 hover:text-accent">
          Open <ExternalLink className="size-3" />
        </a>
      </figcaption>
    </figure>
  );
}
