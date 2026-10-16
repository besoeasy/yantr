import { computed, ref } from "vue";
import {
  COMMENT_KIND,
  COMMENT_MAX_LENGTH,
  NOSTR_COMMENT_RELAYS,
  appCommentTag,
  buildCommentTags,
  commentFilter,
  hasValidId,
  hexToNpub,
  isAppComment,
} from "../utils/nostr";
import { developerShortKey } from "../utils/developer";

// Everything the app-page comments section needs: one relay fan-out for
// reading, and the reader's own NIP-07 signer for writing.
//
// Comments live on public relays, never on this host. Reading opens one
// NIP-01 REQ per relay and merges what arrives; publishing signs locally
// through `window.nostr` (Alby, nos2x, …) and broadcasts the signed event.
// No key material touches Yantr either way.
//
// A factory rather than module-level state, like useDeveloperProfile: the app
// id changes on every navigation and two app pages must not share a list.

// QUERY_TIMEOUT bounds one relay read: dial, REQ, EVENTs and EOSE. A relay
// that accepts the socket and then says nothing is the failure this exists
// for.
const QUERY_TIMEOUT_MS = 8000;

// queryOneRelay runs a single REQ/EVENT/EOSE exchange against one relay and
// resolves with whatever comment-shaped events arrived. It never rejects: an
// unreachable relay means fewer comments, not a failed page.
function queryOneRelay(relay, filter, subId) {
  return new Promise((resolve) => {
    const found = [];
    let settled = false;
    let reached = false;

    const done = () => {
      if (!settled) {
        settled = true;
        try {
          ws.close();
        } catch {
          // Already gone; the events collected so far still count.
        }
        resolve({ events: found, reached });
      }
    };

    let ws;
    try {
      ws = new WebSocket(relay);
    } catch {
      resolve(found);
      return;
    }

    const timer = setTimeout(done, QUERY_TIMEOUT_MS);

    ws.onopen = () => {
      reached = true;
      try {
        ws.send(JSON.stringify(["REQ", subId, filter]));
      } catch {
        clearTimeout(timer);
        done();
      }
    };

    ws.onmessage = (ev) => {
      let msg;
      try {
        msg = JSON.parse(ev.data);
      } catch {
        return;
      }
      if (!Array.isArray(msg) || msg.length === 0) return;

      const [verb, ...rest] = msg;
      if (verb === "EVENT" && rest[0] === subId && rest[1]?.kind === COMMENT_KIND) {
        found.push(rest[1]);
      } else if (verb === "EOSE" && rest[0] === subId) {
        // Unsubscribe before leaving, so a relay that keeps subscriptions
        // alive does not hold resources for a socket we drop anyway.
        try {
          ws.send(JSON.stringify(["CLOSE", subId]));
        } catch {
          // Closing anyway; the events already delivered still count.
        }
        clearTimeout(timer);
        done();
      }
      // NOTICE / CLOSED / AUTH frames carry no comments and need no handling.
    };

    ws.onerror = () => {
      clearTimeout(timer);
      done();
    };
    ws.onclose = () => {
      clearTimeout(timer);
      done();
    };
  });
}

// publishToRelay broadcasts one signed event and resolves true on a positive
// NIP-20 OK. Auth-gated relays answer with AUTH or a false OK; both are a
// "no" from this relay rather than a failure of the publish.
function publishToRelay(relay, signedEvent) {
  return new Promise((resolve) => {
    let settled = false;
    const done = (ok) => {
      if (!settled) {
        settled = true;
        try {
          ws.close();
        } catch {
          // The verdict is already recorded; a close failure changes nothing.
        }
        resolve(ok);
      }
    };

    let ws;
    try {
      ws = new WebSocket(relay);
    } catch {
      resolve(false);
      return;
    }

    const timer = setTimeout(() => done(false), QUERY_TIMEOUT_MS);

    ws.onopen = () => {
      try {
        ws.send(JSON.stringify(["EVENT", signedEvent]));
      } catch {
        clearTimeout(timer);
        done(false);
      }
    };
    ws.onmessage = (ev) => {
      let msg;
      try {
        msg = JSON.parse(ev.data);
      } catch {
        return;
      }
      if (Array.isArray(msg) && msg[0] === "OK" && msg[1] === signedEvent.id) {
        clearTimeout(timer);
        done(msg[2] === true);
      }
    };
    ws.onerror = () => {
      clearTimeout(timer);
      done(false);
    };
    ws.onclose = () => {
      clearTimeout(timer);
      done(false);
    };
  });
}

export function useAppComments(appId) {
  const id = computed(() => String(appId ?? "").trim().toLowerCase());

  const comments = ref([]);
  const loading = ref(false);
  const loadFailed = ref(false);
  const relaysReached = ref(0);
  const relaysTotal = ref(NOSTR_COMMENT_RELAYS.length);

  const publishing = ref(false);
  const publishError = ref("");
  const justPublishedId = ref("");

  // The connected NIP-07 account, as hex from the signer. Null until the
  // reader connects; the composer renders the connect button instead.
  const pubkeyHex = ref(null);
  const signerMissing = ref(false);

  const npub = computed(() => (pubkeyHex.value ? hexToNpub(pubkeyHex.value) : ""));
  const shortAuthor = computed(() =>
    npub.value ? developerShortKey(npub.value) : ""
  );
  const sortedComments = computed(() =>
    [...comments.value].sort((a, b) => (b.created_at || 0) - (a.created_at || 0))
  );

  function reset() {
    comments.value = [];
    loadFailed.value = false;
    relaysReached.value = 0;
    publishError.value = "";
    justPublishedId.value = "";
  }

  // load fans the comment filter out to every relay and merges what arrives.
  // First answer does not win here, unlike the kind-0 profile fetch: comments
  // are a set, and every relay may hold ones the others do not.
  async function load() {
    if (!id.value) return;
    reset();
    loading.value = true;

    const subId = `yantr-comments-${id.value}-${Date.now().toString(36)}`;
    const filter = commentFilter(id.value);
    const nowSec = Math.floor(Date.now() / 1000);

    const perRelay = await Promise.all(
      NOSTR_COMMENT_RELAYS.map((relay) => queryOneRelay(relay, filter, subId))
    );

    relaysReached.value = perRelay.filter((r) => r.reached).length;

    const seen = new Set();
    const merged = [];
    for (const { events } of perRelay) {
      for (const event of events || []) {
        if (!event || typeof event.id !== "string" || seen.has(event.id)) continue;
        seen.add(event.id);
        if (isAppComment(event, id.value, nowSec)) merged.push(event);
      }
    }

    // Best-effort id check, one event at a time: a mismatch drops the one
    // event, never the list. Skipped when WebCrypto is unavailable.
    if (typeof crypto !== "undefined" && crypto.subtle) {
      const verified = [];
      for (const event of merged) {
        try {
          if (await hasValidId(event)) verified.push(event);
        } catch {
          // An unverifiable event is dropped, not fatal.
        }
      }
      comments.value = verified;
    } else {
      comments.value = merged;
    }

    // No relay reachable is the only hard failure: an empty list with relays
    // answering is a real answer — nobody commented yet.
    loadFailed.value = relaysReached.value === 0;
    loading.value = false;
  }

  function hasSigner() {
    return (
      typeof window !== "undefined" &&
      !!window.nostr &&
      typeof window.nostr.getPublicKey === "function" &&
      typeof window.nostr.signEvent === "function"
    );
  }

  // connect asks the NIP-07 extension for the account that will sign. The
  // public key is the only thing that crosses into the page; signing stays
  // inside the extension.
  async function connect() {
    signerMissing.value = false;
    publishError.value = "";
    if (!hasSigner()) {
      signerMissing.value = true;
      return false;
    }
    try {
      const hex = await window.nostr.getPublicKey();
      if (!hex || !/^[0-9a-fA-F]{64}$/.test(String(hex))) {
        publishError.value = "signer-error";
        return false;
      }
      pubkeyHex.value = String(hex).toLowerCase();
      return true;
    } catch {
      publishError.value = "signer-error";
      return false;
    }
  }

  // publish signs one comment with the reader's extension and broadcasts it.
  // One positive OK is success: relays differ in auth policy and retention,
  // and unanimity was never on offer.
  async function publish(text) {
    publishError.value = "";
    justPublishedId.value = "";

    const content = String(text ?? "").trim();
    if (!content) {
      publishError.value = "empty";
      return false;
    }
    if (content.length > COMMENT_MAX_LENGTH) {
      publishError.value = "too-long";
      return false;
    }

    if (!pubkeyHex.value) {
      const ok = await connect();
      if (!ok) return false;
    }

    publishing.value = true;
    try {
      const created_at = Math.floor(Date.now() / 1000);
      const unsigned = {
        kind: COMMENT_KIND,
        pubkey: pubkeyHex.value,
        created_at,
        tags: buildCommentTags(id.value, created_at),
        content,
      };

      let signed;
      try {
        signed = await window.nostr.signEvent(unsigned);
      } catch {
        publishError.value = "sign-cancelled";
        return false;
      }
      if (!signed || typeof signed.id !== "string" || typeof signed.sig !== "string") {
        publishError.value = "sign-error";
        return false;
      }

      const oks = await Promise.all(
        NOSTR_COMMENT_RELAYS.map((relay) => publishToRelay(relay, signed))
      );
      if (!oks.some(Boolean)) {
        publishError.value = "publish-failed";
        return false;
      }

      comments.value = [
        { ...signed, content: signed.content ?? content },
        ...comments.value,
      ];
      justPublishedId.value = signed.id;
      return true;
    } finally {
      publishing.value = false;
    }
  }

  function authorNpub(event) {
    return hexToNpub(event?.pubkey);
  }

  function eventTag() {
    return appCommentTag(id.value);
  }

  return {
    comments: sortedComments,
    loading,
    loadFailed,
    relaysReached,
    relaysTotal,
    publishing,
    publishError,
    justPublishedId,
    pubkeyHex,
    signerMissing,
    npub,
    shortAuthor,
    maxLength: COMMENT_MAX_LENGTH,
    load,
    connect,
    publish,
    authorNpub,
    eventTag,
  };
}
