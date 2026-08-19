# Cineplex CLI — reverse-engineering notes

Cineplex ships no public API. This CLI was generated from an internal-YAML spec built by
capturing the cineplex.com front end's own traffic (a Next.js app over an Azure APIM gateway
at `apis.cineplex.com`, backed by Vista cinema software).

## Auth model (two systems)
1. **`apis.cineplex.com`** — header `Ocp-Apim-Subscription-Key` (a public, anonymous key the
   site ships to every browser). Set it via `CINEPLEX_SUBSCRIPTION_KEY`. Read endpoints and
   seat *layout* work with this alone.
2. **SCENE+ session** — authed ticketing (cart, seat *availability*, reserve, points) needs a
   user session token minted from a logged-in SCENE+ session
   (`/prod/ticketing/api/v1/loyalty/token`), and the `connect.cineplex.com/Account/CCWebConnect/*`
   account endpoints use a separate cookie session. Login is email + password + **SMS OTP**.

## Endpoint confidence
- **CONFIRMED live** (verified against the real API): theatres nearby/detail, movies
  now-playing/coming-soon/detail, showtimes list + detail, experiences, seat-layout,
  booking-fees. These return real data with just the subscription key.
- **UNVERIFIED request bodies**: order/cart mutations (`ticketing-cart`), `reserve-seats`,
  concessions, voucher, and payment `init`/`confirm`. Paths are confirmed from captured
  traffic; the exact JSON bodies still need a full logged-in capture and are marked
  `# UNVERIFIED` in the spec.

## Novel commands (hand-authored, not generator-emitted)
- `showtimes best` — rank sessions across theatres by target time (`--near`), window
  (`--after`/`--before`), movie substring, and experience filter; scores by closeness to the
  target time then seats remaining, and exposes a format-quality rank.

## Payment reality
Final card payment is a PCI-DSS hosted form + 3-D Secure — it cannot be automated headless.
Cart building, seat reservation, and SCENE+ points/gift-card redemption are API-driven; only
the card + 3DS step is browser-bound.

## Not yet built
- `auth login` (SCENE+ email/password/OTP → session token). Needed before live seat
  availability, reserve, cart, and account commands work end to end.
