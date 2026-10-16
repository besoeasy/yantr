import qrcode from "qrcode-generator";

// QR symbol generation, kept out of the component so it can be exercised without
// a DOM — which is the only test environment this project has. Encoding, sizing
// and drawing are three separate concerns and only the last one needs Vue.

// The clear margin the specification requires on every side, in modules. Four is
// the minimum; anything less and many readers refuse to lock on.
export const QR_QUIET_ZONE = 4;

// Error correction level M recovers roughly 15% of the symbol. Level L would give
// a slightly smaller code, but this one is read off a screen, where glare and
// focus are the likely problems, and the payload is short enough that the extra
// modules cost almost nothing.
const QR_ERROR_CORRECTION = "M";

// buildQrSymbol encodes one payload and returns what is needed to draw it, or
// null when the payload cannot be encoded at all.
//
// The path is a single unit square per dark module, in a coordinate space whose
// origin is the top-left module. A caller draws it in a viewBox starting at
// -QR_QUIET_ZONE so the quiet zone is painted rather than inherited from
// whatever is behind the code.
export function buildQrSymbol(payload) {
  const value = String(payload ?? "");
  if (!value) return null;

  try {
    // Type 0 asks for the smallest symbol the payload fits in.
    const qr = qrcode(0, QR_ERROR_CORRECTION);
    qr.addData(value);
    qr.make();

    const count = qr.getModuleCount();
    let path = "";
    for (let row = 0; row < count; row++) {
      for (let col = 0; col < count; col++) {
        if (!qr.isDark(row, col)) continue;
        path += `M${col} ${row}h1v1h-1z`;
      }
    }

    return { path, count, extent: count + QR_QUIET_ZONE * 2 };
  } catch {
    // The encoder throws when a payload fits no symbol at all. A key that long
    // is a catalog mistake, and it must not take the page down with it.
    return null;
  }
}
