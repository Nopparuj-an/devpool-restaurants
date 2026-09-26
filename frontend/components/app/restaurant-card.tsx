import Link from "next/link";

import { Badge, Photo, Rating } from "@/components/ui/misc";
import type { RestaurantSummary } from "@/lib/types";

export function RestaurantCard({ restaurant: r, limited }: { restaurant: RestaurantSummary; limited?: boolean }) {
  return (
    <Link href={`/restaurants/${r.id}`} className="group flex flex-col gap-3">
      <div className="relative overflow-hidden rounded-xl">
        <Photo src={r.cover_url} className="aspect-[3/2] w-full transition-transform duration-300 group-hover:scale-[1.02]" />
        {limited && (
          <span className="absolute left-3 top-3">
            <Badge tone="warning">Limited seats left</Badge>
          </span>
        )}
      </div>
      <div className="flex flex-col gap-1">
        <h3 className="font-medium leading-snug group-hover:text-accent">{r.name}</h3>
        <p className="text-sm text-muted">
          {r.cuisine} · {r.location}
        </p>
        <Rating value={r.rating} count={r.review_count} />
      </div>
    </Link>
  );
}
