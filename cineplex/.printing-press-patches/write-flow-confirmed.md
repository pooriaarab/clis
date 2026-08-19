# Cineplex write/checkout flow — CONFIRMED by monitoring a real purchase (2026-08-18)
Real purchase completed (confirmation WL35LK7, theatre 1151 Park Royal). Contracts captured (values redacted):

## Auth for authed ticketing = SCENE+ SESSION COOKIE (not a bearer token)
Availability/cart/reserve return 200 in-browser with only `Ocp-Apim-Subscription-Key` header PLUS the SCENE+
session COOKIE (sent automatically by the browser). No `Authorization: Bearer` header is used. The CLI's
`auth set-token` (bearer) is therefore WRONG for these — it must send the SCENE+ session cookie as `Cookie:`.
CLI needs `auth login` to capture that cookie (browser import) OR `auth set-cookie <value>`.

## Endpoints (apis.cineplex.com/prod/ticketing/api/v1)
- GET theatre/{tid}/showtime/{sid}/seat-availability-for-cart
    -> { "seatAvailabilities": { "<seatId>": "Available" | "Occupied", ... }, "isSoldOut": bool, "isPostShowtime": bool }
    seatId format matches seat-layout seat.id ("row_col_col", e.g. "1_17_30"). JOIN layout.id -> availability status.
- GET  ticketing-cart                          # read current cart
- PATCH ticketing-cart                         # WRITE: set seat selection into the cart (object body w/ seats)
- POST theatre/{tid}/showtime/{sid}/reserve-seats   # WRITE: ~EMPTY body; holds the cart's selected seats
    -> { "value": ..., "cart": {...}, "expiredAt": <hold-expiry> }

## Payment = THIRD-PARTY HOSTED (browser-only, CANNOT be CLI'd)
Cineplex hands off to E-xact / Elavon at https://checkout.e-xact.com/collect_payment_data
  ?ant=...&merchant=WSP-CINEP-...&order=...&purch=...  (PCI-hosted form + 3DS).
Confirmation returns to https://www.cineplex.com/ticketing/payment/confirmation?confirmationId=XXXX&locationId=NNNN

## CLI implications
- Availability-aware `seats best`: fetch seat-layout (id->label) + seat-availability-for-cart (id->status),
  keep only "Available", then rank. Needs the SCENE+ cookie for availability; physical-best without it.
- `seats reserve`: PATCH ticketing-cart (set seats) then POST reserve-seats. Needs the cookie. Reserve body ~empty.
- Payment stays browser-only; CLI flow ends at `seats open` / hand-off to the hosted page.
