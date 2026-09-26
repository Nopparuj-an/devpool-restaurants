"use client";

import { ImagePlus, Star, X } from "lucide-react";
import { useState, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/controls";
import { Field, Input, Select, Textarea } from "@/components/ui/field";
import { Badge, Notice, Photo } from "@/components/ui/misc";
import { WEEKDAYS, duration } from "@/lib/format";
import type { Hours, RestaurantDetail, RestaurantImage } from "@/lib/types";

export type RestaurantInput = {
  name: string;
  description: string;
  cuisine: string;
  location: string;
  seats: number;
  cancel_cutoff_minutes: number;
  max_reservation_minutes: number;
  hours: Hours[];
};

const TIMES: string[] = [];
for (let m = 0; m < 24 * 60; m += 15) {
  TIMES.push(`${String(Math.floor(m / 60)).padStart(2, "0")}:${String(m % 60).padStart(2, "0")}`);
}
const CUTOFFS = [30, 60, 120, 180, 360, 720, 1440];
const MAX_LENGTHS = [60, 90, 120, 180, 240, 360];

type DayRow = { open: boolean; from: string; to: string };

// A photo in the form: either already stored (id) or picked just now (file).
export type PhotoItem = { key: string; url: string; isCover: boolean; id?: number; file?: File };

const MAX_PHOTOS = 10;
const MAX_BYTES = 5 * 1024 * 1024;
const PHOTO_TYPES = ["image/jpeg", "image/png", "image/webp"];

function toRows(hours: Hours[]): DayRow[] {
  return [0, 1, 2, 3, 4, 5, 6].map((d) => {
    const h = hours.find((x) => x.weekday === d);
    return h ? { open: true, from: h.open, to: h.close } : { open: false, from: "11:00", to: "22:00" };
  });
}

function Section({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <section className="grid gap-6 border-b border-line py-8 first:pt-0 last:border-0 md:grid-cols-[14rem_1fr]">
      <div>
        <h2 className="font-medium">{title}</h2>
        {hint && <p className="mt-1 text-sm text-muted">{hint}</p>}
      </div>
      <div className="flex flex-col gap-4">{children}</div>
    </section>
  );
}

// Create or edit a restaurant. Photos: the first one is the cover (R-REST-1).
export function RestaurantForm({
  initial,
  images: initialImages = [],
  onSave,
  onDelete,
}: {
  initial?: RestaurantDetail;
  images?: RestaurantImage[];
  onSave: (input: RestaurantInput, photos: PhotoItem[]) => Promise<{ error?: string }>;
  onDelete?: () => Promise<void>;
}) {
  const [name, setName] = useState(initial?.name ?? "");
  const [cuisine, setCuisine] = useState(initial?.cuisine ?? "");
  const [location, setLocation] = useState(initial?.location ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [seats, setSeats] = useState(initial?.seats ?? 20);
  const [cutoff, setCutoff] = useState(initial?.cancel_cutoff_minutes ?? 30);
  const [maxLength, setMaxLength] = useState(initial?.max_reservation_minutes ?? 240);
  const [rows, setRows] = useState<DayRow[]>(toRows(initial?.hours ?? []));
  const [images, setImages] = useState<PhotoItem[]>(() =>
    initialImages.map((img) => ({ key: `img-${img.id}`, url: img.url, isCover: img.is_cover, id: img.id })),
  );
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [message, setMessage] = useState<{ tone: "success" | "danger"; text: string } | null>(null);

  const setRow = (d: number, patch: Partial<DayRow>) =>
    setRows((rs) => rs.map((r, i) => (i === d ? { ...r, ...patch } : r)));

  async function save() {
    const hours = rows.flatMap((r, weekday) => (r.open ? [{ weekday, open: r.from, close: r.to }] : []));
    if (hours.length === 0) return setMessage({ tone: "danger", text: "Open at least one day a week." });
    if (images.length === 0) return setMessage({ tone: "danger", text: "Add at least one photo." });
    setSaving(true);
    setMessage(null);
    const res = await onSave(
      { name, description, cuisine, location, seats, cancel_cutoff_minutes: cutoff, max_reservation_minutes: maxLength, hours },
      images,
    );
    setSaving(false);
    setMessage(res.error ? { tone: "danger", text: res.error } : { tone: "success", text: "Saved." });
  }

  function addFiles(files: FileList | null) {
    if (!files) return;
    const picked = Array.from(files);
    const bad = picked.find((f) => !PHOTO_TYPES.includes(f.type) || f.size > MAX_BYTES);
    if (bad) return setMessage({ tone: "danger", text: `${bad.name} must be a JPEG, PNG or WebP under 5 MB.` });
    setMessage(null);
    setImages((xs) => {
      const room = MAX_PHOTOS - xs.length;
      const added = picked.slice(0, room).map((file, i) => ({
        key: `new-${crypto.randomUUID()}`,
        url: URL.createObjectURL(file),
        isCover: xs.length === 0 && i === 0,
        file,
      }));
      return [...xs, ...added];
    });
  }

  const hintFor = (r: DayRow) =>
    !r.open ? null : r.from === r.to ? "Open 24 hours" : r.to < r.from ? "Closes the next day" : null;

  return (
    <div className="flex flex-col">
      <Section title="About" hint="What customers see first.">
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={100} />
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Cuisine">
            <Input value={cuisine} onChange={(e) => setCuisine(e.target.value)} placeholder="Thai, Isan, Café…" maxLength={50} />
          </Field>
          <Field label="Location">
            <Input value={location} onChange={(e) => setLocation(e.target.value)} placeholder="Area, city" maxLength={200} />
          </Field>
        </div>
        <Field label="Description">
          <Textarea value={description} onChange={(e) => setDescription(e.target.value)} maxLength={2000} />
        </Field>
      </Section>

      <Section title="Photos" hint="The first photo is the cover. Up to 10, JPEG, PNG or WebP, 5 MB each.">
        <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
          {images.map((img, i) => (
            <div key={img.key} className="group relative">
              <Photo src={img.url} className="aspect-square w-full rounded-lg" />
              {img.isCover ? (
                <span className="absolute left-1.5 top-1.5">
                  <Badge tone="accent">Cover</Badge>
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => setImages((xs) => xs.map((x) => ({ ...x, isCover: x.key === img.key })))}
                  className="absolute left-1.5 top-1.5 hidden items-center gap-1 rounded-full bg-white/90 px-2 py-0.5 text-xs group-hover:flex"
                >
                  <Star className="size-3" /> Make cover
                </button>
              )}
              <button
                type="button"
                aria-label={`Remove photo ${i + 1}`}
                disabled={images.length === 1}
                onClick={() =>
                  setImages((xs) => {
                    const left = xs.filter((x) => x.key !== img.key);
                    if (img.isCover && left[0]) left[0] = { ...left[0], isCover: true };
                    return left;
                  })
                }
                className="absolute right-1.5 top-1.5 flex size-6 items-center justify-center rounded-full bg-white/90 opacity-0 transition-opacity group-hover:opacity-100 disabled:hidden"
              >
                <X className="size-3.5" />
              </button>
            </div>
          ))}
          {images.length < 10 && (
            <label className="flex aspect-square cursor-pointer flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-line text-sm text-muted transition-colors hover:border-accent hover:text-accent">
              <ImagePlus className="size-5" />
              Add photos
              <input
                type="file"
                accept={PHOTO_TYPES.join(",")}
                multiple
                className="sr-only"
                onChange={(e) => {
                  addFiles(e.target.files);
                  e.target.value = "";
                }}
              />
            </label>
          )}
        </div>
      </Section>

      <Section title="Bookings" hint="Seats are counted per person, not per table.">
        <div className="grid gap-4 sm:grid-cols-3">
          <Field label="Seats">
            <Input type="number" min={1} max={1000} value={seats} onChange={(e) => setSeats(Number(e.target.value))} />
          </Field>
          <Field label="Cancel until" hint="Before the booking starts">
            <Select value={cutoff} onChange={(e) => setCutoff(Number(e.target.value))}>
              {CUTOFFS.map((m) => (
                <option key={m} value={m}>
                  {duration(m)}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Longest booking">
            <Select value={maxLength} onChange={(e) => setMaxLength(Number(e.target.value))}>
              {MAX_LENGTHS.map((m) => (
                <option key={m} value={m}>
                  {duration(m)}
                </option>
              ))}
            </Select>
          </Field>
        </div>
      </Section>

      <Section
        title="Opening hours"
        hint="If closing time is earlier than opening time, the shift ends the next day. Same time means open 24 hours."
      >
        <div className="flex flex-col divide-y divide-line rounded-xl border border-line">
          {[1, 2, 3, 4, 5, 6, 0].map((d) => {
            const r = rows[d];
            return (
              <div key={d} className="flex flex-wrap items-center gap-3 px-4 py-2.5">
                <label className="flex w-32 items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={r.open}
                    onChange={(e) => setRow(d, { open: e.target.checked })}
                    className="size-4 accent-accent"
                  />
                  {WEEKDAYS[d]}
                </label>
                {r.open ? (
                  <div className="flex items-center gap-2">
                    <Select className="w-24" value={r.from} onChange={(e) => setRow(d, { from: e.target.value })} aria-label={`${WEEKDAYS[d]} opens`}>
                      {TIMES.map((t) => (
                        <option key={t}>{t}</option>
                      ))}
                    </Select>
                    <span className="text-sm text-muted">to</span>
                    <Select className="w-24" value={r.to} onChange={(e) => setRow(d, { to: e.target.value })} aria-label={`${WEEKDAYS[d]} closes`}>
                      {TIMES.map((t) => (
                        <option key={t}>{t}</option>
                      ))}
                    </Select>
                    {hintFor(r) && <span className="text-xs text-muted">{hintFor(r)}</span>}
                  </div>
                ) : (
                  <span className="text-sm text-faint">Closed</span>
                )}
              </div>
            );
          })}
        </div>
      </Section>

      <div className="flex flex-wrap items-center gap-3 pt-8">
        <Button onClick={save} disabled={saving}>
          {saving ? "Saving…" : initial ? "Save changes" : "Create restaurant"}
        </Button>
        {message && <Notice tone={message.tone}>{message.text}</Notice>}
        {initial && onDelete && (
          <Button variant="danger" className="ml-auto" onClick={() => setConfirmDelete(true)}>
            Delete restaurant
          </Button>
        )}
      </div>

      {initial && onDelete && (
        <ConfirmDialog
          open={confirmDelete}
          title={`Delete ${initial.name}?`}
          confirmLabel="Delete"
          danger
          onClose={() => setConfirmDelete(false)}
          onConfirm={async () => {
            setConfirmDelete(false);
            await onDelete();
          }}
        >
          {initial.upcoming_reservations ? (
            <>
              This also deletes <span className="font-medium text-ink">{initial.upcoming_reservations} upcoming bookings</span> and
              every review. You can&apos;t undo this.
            </>
          ) : (
            <>This deletes the restaurant and its reviews. You can&apos;t undo this.</>
          )}
        </ConfirmDialog>
      )}
    </div>
  );
}
