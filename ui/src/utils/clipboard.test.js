/**
 * clipboard.test.js — the two paths a copy can take.
 *
 * The bug behind these tests: the container page copied the shell command with
 * `navigator.clipboard.writeText()` alone, and reported "Could not copy the
 * shell command" whenever the page was opened over plain HTTP — which is how
 * Yantr is usually opened, since the backend serves no TLS. `navigator.clipboard`
 * is simply absent there, so the call threw before any permission was involved.
 *
 * Vitest runs these in a Node environment, so the DOM is a stub: only the
 * members `copyText` actually touches are modelled, and a missing one would
 * fail loudly rather than silently pass.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { copyText } from "./clipboard.js";

function fakeDocument({ execCommandResult = true, execCommandThrows = false } = {}) {
  const body = {
    appended: [],
    appendChild(el) {
      this.appended.push(el);
    },
    removeChild(el) {
      this.appended = this.appended.filter((child) => child !== el);
    },
  };

  const selection = {
    rangeCount: 0,
    removeAllRanges: vi.fn(),
    addRange: vi.fn(),
  };

  const created = [];

  const document = {
    body,
    getSelection: () => selection,
    createElement: (tag) => {
      const element = {
        tag,
        value: "",
        style: {},
        attributes: {},
        setAttribute(name, value) {
          this.attributes[name] = value;
        },
        select: vi.fn(),
      };
      created.push(element);
      return element;
    },
    execCommand: vi.fn(() => {
      if (execCommandThrows) throw new Error("copy refused");
      return execCommandResult;
    }),
  };

  return { document, body, selection, created };
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("copyText", () => {
  it("uses the async clipboard API when the page is a secure context", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });

    await expect(copyText("podman exec -it abc bash")).resolves.toBe(true);
    expect(writeText).toHaveBeenCalledWith("podman exec -it abc bash");
  });

  it("falls back to a textarea when the clipboard API is absent", async () => {
    const { document, body } = fakeDocument();
    vi.stubGlobal("navigator", {});
    vi.stubGlobal("document", document);

    await expect(copyText("podman exec -it abc sh")).resolves.toBe(true);
    expect(document.execCommand).toHaveBeenCalledWith("copy");
    // The temporary textarea is removed again, whatever the outcome.
    expect(body.appended).toHaveLength(0);
  });

  it("falls back when the clipboard API exists but refuses the write", async () => {
    const { document } = fakeDocument();
    vi.stubGlobal("navigator", {
      clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) },
    });
    vi.stubGlobal("document", document);

    await expect(copyText("podman exec -it abc bash")).resolves.toBe(true);
    expect(document.execCommand).toHaveBeenCalledWith("copy");
  });

  it("puts the copied text on the textarea it selects", async () => {
    const { document, created } = fakeDocument();
    vi.stubGlobal("navigator", {});
    vi.stubGlobal("document", document);

    await copyText("podman exec -it abc zsh");

    // The value is what gets selected, so it is what lands on the clipboard.
    expect(created).toHaveLength(1);
    expect(created[0].value).toBe("podman exec -it abc zsh");
    expect(created[0].select).toHaveBeenCalledTimes(1);
  });

  it("reports failure instead of throwing when both paths fail", async () => {
    const { document } = fakeDocument({ execCommandThrows: true });
    vi.stubGlobal("navigator", {});
    vi.stubGlobal("document", document);

    await expect(copyText("podman exec -it abc bash")).resolves.toBe(false);
  });

  it("reports failure when execCommand refuses without throwing", async () => {
    const { document } = fakeDocument({ execCommandResult: false });
    vi.stubGlobal("navigator", {});
    vi.stubGlobal("document", document);

    await expect(copyText("podman exec -it abc bash")).resolves.toBe(false);
  });

  it("does nothing for empty text", async () => {
    const writeText = vi.fn();
    vi.stubGlobal("navigator", { clipboard: { writeText } });

    await expect(copyText("")).resolves.toBe(false);
    expect(writeText).not.toHaveBeenCalled();
  });
});
