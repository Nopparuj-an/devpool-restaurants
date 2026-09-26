# 0005 — Garage (S3-compatible) for restaurant images

**Status:** Accepted (2026-09-26; read path confirmed same day)

## Context
Restaurants need a cover image and extras (gallery). Storing blobs in Postgres or on the API container's disk is simple but doesn't match how production apps work. MinIO is the usual choice, but its community edition has become less friendly. Garage is a lightweight, self-hostable S3-compatible store.

## Decision
- Run Garage in docker compose with one bucket, `restaurant-images`.
- **Upload (proposed):** the browser sends multipart to the Go API. Go validates type (jpeg/png/webp), size (e.g. ≤ 5 MB), and ownership, then puts the object via the AWS SDK for Go v2 (custom endpoint, path-style addressing). Alternative: presigned PUT URLs. That is faster, but validation is harder.
- **Read:** serve images **publicly through Garage's web endpoint** (bucket website mode), exposed on the same origin under `/images/*`. Restaurant photos are public content, so access control buys nothing. Stable URLs cache well in the browser and in `next/image`, and list pages don't need a signing call per image.
  - Rejected: presigned GET. URLs expire, change on every request (which breaks caching), and the API has to sign N URLs per list page.
  - Caveat: Garage's web endpoint picks the bucket from the `Host` header (`<bucket>.<root_domain>`). The proxy in front of it must send that Host, or add a network alias in compose. If that gets fiddly, fall back to a tiny Go handler `GET /images/{key}` that streams from S3 with `Cache-Control: public, max-age=…, immutable`.
- Object keys are `restaurants/{restaurant_id}/{uuid}.{ext}`. The DB stores the key, never the full URL.

## Consequences
- Garage needs a **one-time init** (node layout assign + apply, key creation, bucket creation, and granting the key access). This must be scripted in `deployment/` so `docker compose up` works on a fresh clone.
- Deleting a restaurant deletes its objects after the DB commit (best effort, and log failures).
- Seed data must upload demo images.
