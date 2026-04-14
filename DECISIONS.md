# Architecture Decision Records

## 1. gRPC + grpc-gateway

**Decision**: Use gRPC internally with grpc-gateway translating HTTP/JSON at the edge.

**Reasoning**: The browser cannot speak raw gRPC (HTTP/2 trailers not supported). grpc-gateway generates a reverse proxy that accepts standard REST/JSON and forwards to the gRPC server, giving us a clean proto-first contract without a separate REST layer.

---

## 2. No ORM — raw `database/sql` with `lib/pq`

**Decision**: Use Go's standard `database/sql` with the `lib/pq` Postgres driver directly.

**Reasoning**: The query surface is small and well-defined. An ORM adds abstraction overhead and makes conflict detection with `SELECT FOR UPDATE` harder to reason about. Raw SQL is explicit and auditable.

---

## 3. Concurrent booking — `SELECT FOR UPDATE`

**Decision**: Lock all of a user's scheduled appointments inside a transaction before checking for overlaps.

**Reasoning**: Without a row-level lock, two concurrent requests could both pass the conflict check and both insert — leaving the user with a double-booking. `SELECT FOR UPDATE` serialises writes per user, at the cost of throughput under high concurrency for the same user (acceptable for a scheduling product).

---

## 4. Idempotency keys

**Decision**: Store a UUID idempotency key per create request in a separate table. On duplicate key, return the original appointments.

**Reasoning**: Network retries should not create duplicate appointments. The client generates a UUID per submission (via `uuidv4()`). The server stores it with `ON CONFLICT DO NOTHING` and re-fetches the original result if the key was already seen.

---

## 5. JWT in localStorage

**Decision**: Store the JWT in `localStorage` rather than an `HttpOnly` cookie.

**Trade-off**: `localStorage` is accessible to JavaScript, making it vulnerable to XSS. An `HttpOnly` cookie is immune to XSS but requires same-origin deployment and CSRF protection. For this assessment the simplicity of `localStorage` was preferred; in production, `HttpOnly` cookies with `SameSite=Strict` and a short-lived access token + refresh token rotation would be the safer choice.

---

## 6. Timezone handling — browser Intl API only

**Decision**: Use `Intl.DateTimeFormat` and `Intl.supportedValuesOf('timeZone')` for all timezone work. No date-fns or moment.

**Reasoning**: The browser's Intl API is sufficient for display formatting and offset calculation. Adding a timezone library adds bundle weight without benefit when all we need is rendering and positioning on a calendar grid.

**Known gap**: The calendar always positions appointments relative to the timezone stored in the user's profile. If the user travels and their browser timezone shifts, appointment times display in their registered timezone, not the local one. This is intentional — a scheduling tool should show times in a consistent zone.

---

## 7. Recurrence — weekly only, max 4 occurrences

**Decision**: Only weekly recurrence is supported. The recurrence end date is stored, and individual appointment rows are created per occurrence at write time (not expanded at read time).

**Reasoning**: Expanding at write time simplifies queries — no recurrence rule parsing at read time. The four-occurrence cap keeps the insert set bounded. A full iCal RRULE engine was out of scope for this assessment.

---

## 8. React Query for server state

**Decision**: All server data (appointments, profile) is fetched and cached via TanStack Query. No Redux or Zustand.

**Reasoning**: React Query handles loading/error states, caching, and cache invalidation after mutations out of the box. The application state is almost entirely server state, so a client-side store would be redundant.

---

## 9. Split first/last name

**Decision**: Users register with `first_name` and `last_name` as separate fields rather than a single `display_name`.

**Reasoning**: Split names are more flexible for display ("Hello, John"), sorting, and internationalisation. Added via a migration (`000004`) to keep the schema change incremental.
