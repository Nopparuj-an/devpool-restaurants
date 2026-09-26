# Restaurants — Project Wiki

A restaurant reservation and review community. Every account can open its own restaurants (as **owner**) and book or review anyone's restaurant (as **customer**). This is the PEA DevPool 2026 final exam project.

This wiki is the source of truth for humans and coding agents. If the code and the wiki disagree, one of them is a bug. Fix whichever is wrong in the same change.

## Pages

| Page | What it answers |
|---|---|
| [requirements.md](requirements.md) | What the exam asks for, deadlines, deliverables |
| [domain.md](domain.md) | Glossary and **business rules** (booking, cancel, reviews, hours) |
| [architecture.md](architecture.md) | Stack, components, auth flow, data model, API conventions |
| [decisions/](decisions/) | Architecture Decision Records (why things are the way they are) |
| [roadmap.md](roadmap.md) | Milestones and progress checklist |
| [open-questions.md](open-questions.md) | Things not decided yet |
| [interview-prep.md](interview-prep.md) | Questions the interview will likely ask, with our answers |

## Repo layout

```
backend/     Go API
frontend/    Next.js app
deployment/  docker compose, Garage config, seed scripts
docs/        this wiki
```

## Conventions for editing this wiki

- Business rules live in `domain.md` only. Other pages link to them instead of restating them.
- A decision that constrains future work gets an ADR in `decisions/` (`NNNN-short-title.md`). Don't rewrite old ADRs. When one is replaced, add a new ADR and mark the old one `Superseded by NNNN`.
- When a question in `open-questions.md` is settled, move the answer to the right page (or an ADR) and delete the question.
- Tick items in `roadmap.md` as they land.

## For agents

1. Read `domain.md` before touching booking, review, or restaurant logic.
2. Enforce every rule in `domain.md` in the Go backend. The frontend may check the same rules again for UX, but it is never the only place a rule is enforced.
3. Don't make product decisions silently. If a rule is missing, add it to `open-questions.md` and ask.
