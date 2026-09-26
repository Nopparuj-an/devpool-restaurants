// Placeholder until the real UI lands (see docs/roadmap.md). It renders the
// restaurant list on the server to prove web → API → images wiring works.

type Summary = {
  id: number;
  name: string;
  cuisine: string;
  rating: number | null;
  review_count: number;
  cover_url: string;
};

// Server-side calls go straight to the API (read at request time).
const API_URL = process.env.API_URL ?? "http://localhost:8080";

async function getRestaurants(): Promise<Summary[]> {
  const res = await fetch(`${API_URL}/api/restaurants`, { cache: "no-store" });
  if (!res.ok) throw new Error(`API ${res.status}`);
  return (await res.json()).restaurants;
}

export default async function Home() {
  const restaurants = await getRestaurants();
  return (
    <main className="mx-auto max-w-3xl p-6">
      <h1 className="text-2xl font-semibold">Restaurants</h1>
      <p className="mb-6 text-sm text-neutral-500">Frontend coming soon, wiring check only.</p>
      <ul className="grid gap-4 sm:grid-cols-2">
        {restaurants.map((r) => (
          <li key={r.id} className="overflow-hidden rounded-lg border border-neutral-200">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={r.cover_url} alt="" className="aspect-[3/2] w-full object-cover" />
            <div className="p-3">
              <div className="font-medium">{r.name}</div>
              <div className="text-sm text-neutral-500">
                {r.cuisine} · {r.rating === null ? "No reviews yet" : `★ ${r.rating.toFixed(1)} (${r.review_count})`}
              </div>
            </div>
          </li>
        ))}
      </ul>
    </main>
  );
}
