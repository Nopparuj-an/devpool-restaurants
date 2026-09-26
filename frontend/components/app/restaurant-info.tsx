"use client";

import { WEEKDAYS, shift } from "@/lib/format";
import { Photo } from "@/components/ui/misc";
import type { Hours, RestaurantImage } from "@/lib/types";

// Cover plus up to two extra photos. Phones show the cover only.
export function Gallery({ images, name }: { images: RestaurantImage[]; name: string }) {
  const [cover, ...rest] = [...images].sort((a, b) => Number(b.is_cover) - Number(a.is_cover));
  if (!cover) return <Photo src="" className="aspect-[2/1] w-full rounded-xl" />;
  const extras = rest.slice(0, 2);
  // Fixed height on wider screens; photos fill their cells with object-cover.
  const cell = "relative overflow-hidden rounded-xl bg-surface";
  const fill = "absolute inset-0 h-full w-full";
  return (
    <div className={`grid gap-2 sm:h-80 ${extras.length ? "sm:grid-cols-[2fr_1fr]" : ""}`}>
      <div className={`${cell} aspect-[3/2] sm:aspect-auto`}>
        <Photo src={cover.url} alt={name} className={fill} />
      </div>
      {extras.length > 0 && (
        <div className="hidden min-h-0 gap-2 sm:grid" style={{ gridTemplateRows: `repeat(${extras.length}, 1fr)` }}>
          {extras.map((img) => (
            <div key={img.id} className={cell}>
              <Photo src={img.url} className={fill} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// Monday first; days without a shift show as closed.
export function HoursList({ hours }: { hours: Hours[] }) {
  const byDay = new Map(hours.map((h) => [h.weekday, h]));
  return (
    <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1.5 text-sm">
      {[1, 2, 3, 4, 5, 6, 0].map((d) => {
        const h = byDay.get(d);
        return (
          <div key={d} className="contents">
            <dt className="text-muted">{WEEKDAYS[d]}</dt>
            <dd className={h ? "" : "text-faint"}>{h ? shift(h.open, h.close) : "Closed"}</dd>
          </div>
        );
      })}
    </dl>
  );
}
