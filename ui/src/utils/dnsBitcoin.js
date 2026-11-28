// The donate card's Bitcoin address source: a DNS TXT record read over
// DNS-over-HTTPS, kept out of the component so it can be exercised without a
// network or a mounted Vue tree. The widget's Retry re-calls fetchBitcoinAddress
// after a rejected attempt; a transient failure must not poison later tries.

export const BTC_TXT_NAME = "btc.besoeasy.com";
export const DOH_URL =
  `https://cloudflare-dns.com/dns-query?name=${BTC_TXT_NAME}&type=TXT`;

// Cloudflare DoH returns TXT `data` as one or more quoted strings; a record
// longer than 255 bytes is split into adjacent `"…" "…"` chunks that must be
// concatenated, and surrounding quotes stripped. Multiple Answer entries are
// joined too. Returns "" when nothing usable came back.
export function parseTxtAnswers(data) {
  const answers = Array.isArray(data?.Answer) ? data.Answer : [];
  return answers
    .map((a) => (typeof a?.data === "string" ? a.data : ""))
    .join("")
    .replace(/^"|"$/g, "")
    .replace(/"\s*"/g, "")
    .trim();
}

// Fetch + parse. `fetchImpl` defaults to the global fetch and is injectable so
// callers (and tests) can stand in a stub. Rejects on transport/HTTP error or
// on an empty record; the caller owns retry.
export async function fetchBitcoinAddress(fetchImpl = globalThis.fetch) {
  const res = await fetchImpl(DOH_URL, { headers: { Accept: "application/dns-json" } });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const data = await res.json();
  const raw = parseTxtAnswers(data);
  if (!raw) throw new Error("No TXT record returned");
  return raw;
}
