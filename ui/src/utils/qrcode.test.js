import { describe, expect, it } from "vitest";
import jsQR from "jsqr";
import { buildQrSymbol, QR_QUIET_ZONE } from "./qrcode";

// ─── Drawing data ─────────────────────────────────────────────────────────────

// buildQrSymbol emits one "M<col> <row>h1v1h-1z" subpath per dark module. The
// tests parse that format rather than re-deriving the modules from the encoder,
// so a truncated or malformed path fails here instead of shipping.
const MODULE_RE = /M(-?\d+) (-?\d+)h1v1h-1z/g;

function darkModules(symbol) {
  return [...symbol.path.matchAll(MODULE_RE)].map((m) => ({
    col: Number(m[1]),
    row: Number(m[2]),
  }));
}

// Rasterises the drawing data into an RGBA buffer with a SCALE-pixel module, so
// an independent decoder can read it back.
function rasterise(symbol, scale = 8) {
  const pixels = symbol.extent * scale;
  const buffer = new Uint8ClampedArray(pixels * pixels * 4).fill(255);

  for (const { col, row } of darkModules(symbol)) {
    for (let y = 0; y < scale; y++) {
      for (let x = 0; x < scale; x++) {
        const offset = (((row + QR_QUIET_ZONE) * scale + y) * pixels +
          (col + QR_QUIET_ZONE) * scale + x) * 4;
        buffer[offset] = 0;
        buffer[offset + 1] = 0;
        buffer[offset + 2] = 0;
        buffer[offset + 3] = 255;
      }
    }
  }

  return { buffer, pixels };
}

const NPUB = "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6";
const NPROFILE =
  "nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpz3mhxue69uhhyetvv9ujuerpd46hxtnfduqs6amnwvaz7tmwdaejumr0dsfg70ls";

// The one thing that actually matters: something else can read the code. This is
// worth a real decoder rather than a shape assertion, because a QR code that
// looks right and scans to nothing is the failure nobody notices — it is the
// developer's visitor who finds out, on a phone, with no error message.
describe("buildQrSymbol", () => {
  it.each([
    ["an npub", `nostr:${NPUB}`],
    ["an nprofile", `nostr:${NPROFILE}`],
    ["a longer payload", `nostr:${"npub1" + "q".repeat(70)}`],
  ])("encodes %s readably", (_label, payload) => {
    const symbol = buildQrSymbol(payload);
    expect(symbol).not.toBeNull();

    const { buffer, pixels } = rasterise(symbol);
    const decoded = jsQR(buffer, pixels, pixels);

    expect(decoded?.data).toBe(payload);
  });

  it("keeps the quiet zone clear of modules", () => {
    const symbol = buildQrSymbol(`nostr:${NPUB}`);

    // The viewBox starts at -QR_QUIET_ZONE and spans extent, so the reported
    // extent has to cover both margins.
    expect(symbol.extent).toBe(symbol.count + QR_QUIET_ZONE * 2);

    for (const { col, row } of darkModules(symbol)) {
      expect(col).toBeGreaterThanOrEqual(0);
      expect(row).toBeGreaterThanOrEqual(0);
      expect(col).toBeLessThan(symbol.count);
      expect(row).toBeLessThan(symbol.count);
    }
  });

  it("produces a valid QR symbol size", () => {
    // Version 1 is 21 modules and each later version adds 4.
    for (const payload of [`nostr:${NPUB}`, `nostr:${NPROFILE}`]) {
      const { count } = buildQrSymbol(payload);
      expect(count).toBeGreaterThanOrEqual(21);
      expect(count).toBeLessThanOrEqual(177);
      expect((count - 21) % 4).toBe(0);
    }
  });

  it("draws different symbols for different payloads", () => {
    expect(buildQrSymbol("nostr:aaa").path).not.toBe(buildQrSymbol("nostr:bbb").path);
  });

  // A key the encoder cannot fit, or an absent one, must not throw: the caller
  // hides the code and the rest of the page is unaffected.
  it("returns null for payloads it cannot encode", () => {
    expect(buildQrSymbol("")).toBeNull();
    expect(buildQrSymbol(null)).toBeNull();
    expect(buildQrSymbol(undefined)).toBeNull();
    expect(buildQrSymbol("x".repeat(5000))).toBeNull();
  });
});
