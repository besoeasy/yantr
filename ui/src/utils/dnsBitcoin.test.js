import { describe, expect, it, vi } from "vitest";
import { fetchBitcoinAddress, parseTxtAnswers, DOH_URL } from "./dnsBitcoin";

const ADDRESS = "bc1qxyz0example000000000000000000000000001";

function okFetch(data) {
  return vi.fn(async () => ({ ok: true, json: async () => data }));
}

describe("parseTxtAnswers", () => {
  it("strips quotes and concatenates split TXT chunks", () => {
    expect(parseTxtAnswers({ Answer: [{ data: `"${ADDRESS}"` }] })).toBe(ADDRESS);
    expect(
      parseTxtAnswers({ Answer: [{ data: `"${ADDRESS.slice(0, 20)}" "${ADDRESS.slice(20)}"` }] })
    ).toBe(ADDRESS);
    expect(parseTxtAnswers({ Answer: [{ data: `"${ADDRESS.slice(0, 20)}"` }, { data: `"${ADDRESS.slice(20)}"` }] })).toBe(ADDRESS);
  });

  it("returns empty string for missing / non-string answers", () => {
    expect(parseTxtAnswers({})).toBe("");
    expect(parseTxtAnswers({ Answer: [] })).toBe("");
    expect(parseTxtAnswers({ Answer: [{ data: 123 }] })).toBe("");
    expect(parseTxtAnswers(null)).toBe("");
  });
});

describe("fetchBitcoinAddress", () => {
  it("queries the project TXT record over DoH and returns the address", async () => {
    const fetchImpl = okFetch({ Answer: [{ data: `"${ADDRESS}"` }] });
    await expect(fetchBitcoinAddress(fetchImpl)).resolves.toBe(ADDRESS);
    expect(fetchImpl).toHaveBeenCalledWith(DOH_URL, { headers: { Accept: "application/dns-json" } });
  });

  it("rejects on HTTP error", async () => {
    const fetchImpl = vi.fn(async () => ({ ok: false, status: 503, json: async () => ({}) }));
    await expect(fetchBitcoinAddress(fetchImpl)).rejects.toThrow("HTTP 503");
  });

  it("rejects on transport error", async () => {
    const fetchImpl = vi.fn(async () => { throw new Error("network down"); });
    await expect(fetchBitcoinAddress(fetchImpl)).rejects.toThrow("network down");
  });

  it("rejects when no TXT record is returned", async () => {
    await expect(fetchBitcoinAddress(okFetch({ Answer: [] }))).rejects.toThrow("No TXT record returned");
  });

  // The widget's Retry simply calls again; a transient first failure must not
  // poison the retry. Mirrors SponsorshipCard.fetchBtc being re-runnable.
  it("error -> retry -> success: a rejected attempt does not block a later call", async () => {
    const fetchImpl = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, status: 500, json: async () => ({}) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ Answer: [{ data: `"${ADDRESS}"` }] }) });

    await expect(fetchBitcoinAddress(fetchImpl)).rejects.toThrow("HTTP 500");
    await expect(fetchBitcoinAddress(fetchImpl)).resolves.toBe(ADDRESS);
    expect(fetchImpl).toHaveBeenCalledTimes(2);
  });
});
