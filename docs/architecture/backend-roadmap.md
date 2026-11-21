# Backend roadmap

The backend roadmap follows the product specification and
`LoVeTea_Technical_Plan_v1.docx`. It is delivered in small, testable stages. A
feature that depends on a product or privacy decision remains explicitly blocked
from production behavior until that decision is made.

## Completed foundation

- **Stages 1-3 — platform, account, and couple:** Go API, PostgreSQL migrations,
  configuration, logging, email/password registration and login, token sessions,
  profiles, invitations, and mutual couple detachment.
- **Stage 4 — onboarding foundation:** private, versioned questionnaires,
  published-test catalog, answer validation, and owner-only answer retrieval.
  The questionnaire content and scoring/recommendation rules have not been
  authored; no assessment or result is fabricated.

## Remaining stages

1. **Stage 5 — home, pet, and synchronization foundation:** establish the
   couple-scoped home API, passive pet state, change versions, idempotent
   operations, and a durable change feed/tombstones before dependent modules.
   The initial implementation includes a home snapshot, passive couple-pet
   appearance state, per-user cursors, per-entity versions, and a filtered change
   feed. `POST /sync/push` supports task, calendar-event, important-date, and
   habit upsert/delete operations, idempotent habit completion changes, plus
   versioned mood and sleep upserts with per-operation idempotency receipts.
   Mood notes remain owner-only in the personal feed. Push support for other
   entity types and expanded home widgets remain.
2. **Stage 6 — planning:** personal and shared tasks, calendar events, important
   dates, recurrence and reminders, with ownership, optimistic concurrency, and
   offline-safe writes. Personal/shared task CRUD, calendar event and important
   date CRUD, ownership checks, optimistic version checks, soft-delete
   tombstones, and sync events are implemented and verified against local
   PostgreSQL. Planning list endpoints now support bounded limit/offset
   pagination. Recurrence expansion, reminder delivery, and request idempotency
   remain.
3. **Stage 7 — wellness and habits:** private-by-default mood and sleep entries,
   personal/shared habits, completions, and summaries with explicit access rules.
   Mood and sleep entries are implemented with optional explicit partner sharing,
   owner-only notes, date-range reads, versioned upserts, and sync events.
   Personal/shared habit CRUD, per-member idempotent completions, weekly streaks,
   date-range summaries, ownership checks, and sync events are implemented and
   verified against local PostgreSQL. Reminder delivery remains.
4. **Stage 8 — content and memories:** moments, media metadata, memory records,
   and short-lived object-storage upload/download authorization. The current
   implementation adds scoped media metadata, size/type-limited MinIO upload
   policies, verified completion, short-lived download URLs, moments with
   same-scope media, and sync events. The plaintext memory API is blocked with
   HTTP 501 until the required E2E protocol, key lifecycle, and safe handling of
   existing plaintext records are designed. An opt-in MinIO integration test
   covers form upload, object verification, download, and cleanup; production
   object-storage lifecycle/retention operations remain. The local Compose
   upload/verify/download/delete integration test now passes.
5. **Stage 9 — communication and notifications:** calm-letter lifecycle, chat
   messages/reactions, notification preferences, durable outbox, and real-time
   delivery. A metadata-only transactional outbox and authenticated WebSocket
   best-effort cursor hints are implemented; clients catch up through
   `/sync/pull` on connect and reconnect.
   Private chat and calm letters remain blocked until the E2E protocol, key
   recovery, multi-device, and revocation behavior are specified and reviewed.
   Notification categories and opt-in defaults must also be decided before push
   delivery is enabled. Do not claim end-to-end encryption before that review.
6. **Stage 10 — games and consent-based wheel:** pair game sessions, scores,
   achievements, and proposed wheel outcomes with explicit acceptance, limits,
   and stop/cancel behavior.
7. **Stage 11 — location and recommendations:** consent-gated location sharing
   with immediate stop and retention controls, plus recommendation APIs that
   suggest rather than commit choices. Location retention and recommendation
   inputs need explicit product decisions before production rollout.
8. **Stage 12 — stabilization and release:** security/privacy review, full
   integration and concurrency testing, sync conflict handling, account export
   and deletion, backup/restore drills, operational health/metrics, and
   deployment/release checks. A CI workflow now runs Go tests, vet, and API
   builds for backend changes.

Each stage must include migrations, authenticated routes, input validation,
ownership enforcement, OpenAPI updates, focused tests, and documentation as
applicable. Sensitive content and personal data are never exposed to a partner
by default. Schema and API details should stay small and module-specific rather
than introducing speculative generic abstractions.

## Execution order and dependencies

The order above is intentional: shared ownership and synchronization primitives
precede offline-capable modules; notifications and the transactional outbox
precede real-time features; media authorization precedes content attachments.
Stages with unresolved product/privacy decisions may be built behind safe,
explicit boundaries, but must not silently assume consent, retention, or
encryption behavior.
