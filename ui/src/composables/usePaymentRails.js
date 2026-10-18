import { computed, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { lnurlEncode } from "../utils/nostr";
import { bip21Uri } from "../utils/bitcoin";

// The payment rails a Nostr account publishes, ready to render: Lightning
// (NIP-57 lud16 → LNURL pay-URL) and on-chain (a Bitcoin address → BIP-21
// URI). One composable because two surfaces print them — the Developer of the
// day card and the developer page — and a rail that validated on one must
// validate identically on the other, down to the QR payload.
//
// Both fields come from the VERIFIED kind-0 (useDeveloperProfile's pipeline),
// and each is validated again here rather than trusted downstream:
//   - the Lightning address must still encode to an LNURL pay-URL;
//   - the Bitcoin address must still pass its checksum (utils/bitcoin.js).
// A rail with no valid payload is simply absent: a QR that scans into a dead
// wallet lookup, or into a mistyped address, is worse than no QR at all.
//
// `prefix` is the i18n namespace holding `lightning`, `onchain`, `qrLabel`,
// `onchainQrLabel`, `copy` and `copied`, so each surface labels the rails in
// its own words.

export function usePaymentRails(profile, prefix) {
  const { t } = useI18n();

  const copiedRail = ref("");
  let copiedTimer = null;

  const rails = computed(() => {
    const out = [];
    const name = String(profile.value?.name ?? "").trim();

    const lud = String(profile.value?.lud16 ?? "").trim();
    const lnurl = lud ? lnurlEncode(lud) : "";
    if (lnurl) {
      out.push({
        kind: "lightning",
        label: t(`${prefix}.lightning`),
        address: lud,
        payload: lnurl,
        qrLabel: t(`${prefix}.qrLabel`, { name }),
      });
    }

    const address = String(profile.value?.bitcoin ?? "").trim();
    const bip21 = bip21Uri(address);
    if (bip21) {
      out.push({
        kind: "onchain",
        label: t(`${prefix}.onchain`),
        address,
        payload: bip21,
        qrLabel: t(`${prefix}.onchainQrLabel`, { name }),
      });
    }

    return out;
  });

  // One address at a time is copied, and the confirmation names which rail it
  // was — with two rails side by side, "Copied" alone leaves the reader
  // guessing what landed on the clipboard.
  async function copyAddress(rail) {
    try {
      await navigator.clipboard.writeText(rail?.address ?? "");
      copiedRail.value = rail.kind;
      if (copiedTimer) clearTimeout(copiedTimer);
      copiedTimer = setTimeout(() => {
        copiedRail.value = "";
      }, 2000);
    } catch {
      // Clipboard needs a secure context; the address is printed next to the
      // button, so there is nothing to recover from.
    }
  }

  // Firing a timer into an unmounted component shows up as a console warning
  // in someone else's session.
  onUnmounted(() => {
    if (copiedTimer) clearTimeout(copiedTimer);
  });

  return { rails, copiedRail, copyAddress };
}