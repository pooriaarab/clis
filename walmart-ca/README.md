# walmart-ca

Read your walmart.ca order history from the terminal without driving a browser.
It replays your own signed-in browser session against walmart.ca's internal
GraphQL API, so the press-and-hold bot check only matters once — at session
export — instead of on every automated run.

## Setup

In signed-in Chrome: DevTools (F12) > Network, refresh
`https://www.walmart.ca/en/orders`, filter for `PurchaseHistoryV2`,
right-click > Copy > Copy as cURL, save to a file. Then:

```
walmart-ca-pp-cli auth import --file curl.txt   # or pipe the capture on stdin
walmart-ca-pp-cli doctor
walmart-ca-pp-cli orders list --limit 50
walmart-ca-pp-cli orders list --search yogurt --json
```

Session lands in `~/.walmart-ca/session.json` (`0600`); values are never
printed. Re-import when calls fail with "access denied" — cookies expire
roughly daily, and the query hash is captured from each import so web-build
rotations are picked up automatically.

## Verified

`auth import`, `doctor`, and the `orders list` request shape against
walmart.ca's live `PurchaseHistoryV2` operation; pagination follows
`pageInfo.nextPageCursor`, fetching pages of up to 20 orders until `--limit`
is reached or the cursor runs out.

## Gaps

- No order-details call yet (per-item prices); the list returns item names
  and quantities per order.
- Sessions cannot self-refresh; re-import when cookies expire.
- Unofficial; personal use only. Not affiliated with Walmart.
