# 0008 — UI design as in-repo React components

**Status:** Accepted (2026-09-26), with a known compliance risk

## Context
The brief marks the UI design tool as **mandatory**: "บังคับ · UI Design — Claude Design หรือ Lovable". Deliverable #3 is "a link or screenshots of the designed screens". We don't want to depend on a proprietary design tool.

## Decision
- Designs are plain React components with mock data, served by the Next.js app itself under a **`/design` route**: a **component gallery** plus **frame previews** of each screen at phone and desktop sizes. There is no separate app, and the design route uses the exact components the real pages use. Look and structure are described in [design.md](../design.md): minimal, white, black text, PEA purple accent.
- They are built **before** the matching Next.js pages, and the real pages reuse the same components. The design and the implementation can't drift apart.
- Deliverable #3 is submitted as screenshots of the gallery plus a link to `design/` in the repo.

## Consequences
- **Risk:** this departs from a mandatory requirement. Mitigation: confirm with the instructors before 2026-10-11. If they insist, generate the same screens in Claude Design as a formality (≈1 hour) and submit both.
- A shared component source means no hand-off step, and the interview story is "design system in code".
- It costs more time than a prompt-to-mockup tool, so keep the screens static and minimal: list, detail + booking, my reservations, my restaurants + form, owner bookings, login.
