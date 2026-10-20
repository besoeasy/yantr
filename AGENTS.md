# Yantr App Authoring

How to add or change an app under `apps/`, and the one rule the dashboard
widgets under `ui/src/Widgets/` live by. This file covers the app recipe
format, the rules for writing one, and §8 — nothing else.

---

## 1. Add an app — checklist

1. Create one folder: `apps/<app-name>/`.
2. Add one `compose.yml` (and an optional `logo.svg`).
3. Fill the `x-yantr` block — see [§3](#3-x-yantr-reference).
4. Write the compose services under the constraints in [§5](#5-compose-constraints).
5. Mark every user-facing port `show: true` — see [§4](#4-ports-and-visibility).
6. Verify:
   ```sh
   node check.js
   cd core && go build ./... && go vet ./... && go test ./...
   ```

---

## 2. File layout

```
apps/<app-name>/
├── compose.yml   # required — the whole app definition
└── logo.svg      # optional — square SVG with a viewBox, auto-detected
```

Not allowed: `info.json`, `Dockerfile`, `entrypoint.sh`, or any build context.
A logo with no `viewBox`/`width`/`height` or a non-square shape still renders,
so `check.js` warns about it rather than failing the app.

---

## 3. `x-yantr` reference

### Required

| Key | Type | Rule |
| --- | --- | --- |
| `name` | string | Display name, free-form (`Open WebUI`); the folder is the id. |
| `tags` | list | 3–5 tags; prefer ones the catalog already uses. |
| `short_description` | string | 50–100 chars. |
| `description` | string | 200–300 chars, written as a YAML `>` block. |
| `usecases` | list | 2–5 entries. |
| `website` | string | Prefer the project's source code repo; second preference is its website. |

### Optional

| Key | Type | Rule |
| --- | --- | --- |
| `notes` | list | Post-deploy hints shown to the user; at most 6. |
| `developers` | list | Nostr `npub1…` public keys — see below. |
| `env_generators` | map | `VAR: {length, charset}`, charset ∈ `alnum` `hex` `numeric` `alpha` `base64url` `alnum_symbols`. |
| `ports` | list | `{port, protocol, show, label, service}`; `protocol` ∈ `HTTP` `HTTPS` `TCP` `UDP` and is required. |

Lists may be written as flow sequences (`[a, b]`) or block sequences; pick
whichever reads better for the entry length.

### `env_generators`

Every `${VAR}` in the compose file that has no default should have a matching
`env_generators` entry, or the deploy form has to supply the value. `check.js`
warns rather than fails, because some values are deliberately typed by the
user (`TUNNEL_TOKEN`) while a secret that must not be blank (`ADMIN_PASSWORD`)
needs a generator. An entry that is never referenced is warned about too, so a
renamed variable does not leave a generator behind.

### `ports`

Port and display metadata lives **only** here — never in compose labels:

```yaml
x-yantr:
  ports:
    - {port: 8080, protocol: HTTP, show: true, label: "Web UI", service: my-app}
```

`service` must name a service that exists in the same file; a dangling one is
dropped at parse time. `protocol` is required on every entry — it does not
default to TCP.

The list is display metadata, not derived from the compose file: `check.js`
does not compare the two, so keep a declared port and its `ports:` entry in
step by hand. An entry repeated for the same port, protocol and service is
rejected.

### `developers` — crediting an app's author

`developers` names who the app is from. It is optional and changes nothing
about deployment; it drives the developer credit on the app page.

```yaml
x-yantr:
  developers: ["npub1hznmntyj254kqhr079a5gt2wvhyll6rz6q67pyjres4lfkql22kq5ml6zh"]
```

- An entry is strictly an `npub1…` key.
- Never an `nsec1…` private key, a `note1…` event id, an `nprofile1…`, or a
  `nostr:` prefix. The loader still tolerates a legacy `nprofile1…` or
  `nostr:`-prefixed entry when it reads an older record, but a recipe must not
  use either form.
- Anything that is not a valid key is dropped when the catalog loads;
  `check.js` rejects it outright, so a typo does not silently uncredit someone.
- The same key twice in one recipe is rejected; list one entry per account.
- Entries are deduplicated on the **decoded** pubkey, not the spelling — the
  same key written in two cases is one developer.
- Credit the **upstream author of the software**, not whoever wrote the recipe.
  Leave the field out when you cannot tie a key to a real account.

---

## 4. Ports and visibility

Every published port is one of two kinds. Decide which before you write it.

| | `show: true` — access port | omitted — work port |
| --- | --- | --- |
| What it is | A UI a browser can open | Protocol plumbing; nothing to click |
| In the UI | Own card with a working **Open** button | Hidden by default; listed under **All** |
| Rule | `protocol: HTTP` or `HTTPS` | Everything else |

`show` is **deny-by-default**: omitting it does not unpublish the port, it only
removes it from the user's default view. The rule to apply is: *if a browser can
show the user something, it is `show: true`; otherwise it is a work port.*

Omit `show` for:

- Any `TCP` / `UDP` port — torrent peers, WireGuard, Syncthing relays, database
  clients, git SSH. A browser cannot open them, so an "Open" button would lie.
- `HTTP` ports that are diagnostics rather than UI — metrics, pprof, health
  checks. Being HTTP is not enough; ask whether a human looks at it.

```yaml
ports:
  - {port: 45001, protocol: HTTP, show: true, label: "Torrent UI", service: torbox}
  - {port: 45000, protocol: HTTP, show: true, label: "File Server", service: torbox}
  - {port: 51413, protocol: TCP, label: "Peer Port", service: torbox}
  - {port: 51413, protocol: UDP, label: "Peer Port", service: torbox}
```

Two consequences:

- **Every port is still catalogued, shown or not.** A work port still
  identifies its service, so it still marks that service as the stack's primary
  one. Dropping the entry instead of omitting `show` would lose that.
- **Host ports are auto-assigned, so hiding a port does not make it
  unreachable** — the user just cannot discover the number. Say so in `notes`
  when a work port matters operationally.

An app may legitimately have zero access ports (a database, a file share). The
automated tests fail an app that publishes ports but marks none `show: true`;
such apps are allow-listed there deliberately.

---

## 5. Compose constraints

1. **Named Podman volumes only** — never bind mounts (rootless SELinux labeling
   requires it).
2. **Any maintained tag** — `:latest` or a pinned release. Pinning a database's
   major version (`postgres:16-alpine`) is a data-compatibility decision, not
   debt; `check.js` requires an image but does not judge its tag.
3. **Prebuilt images only** — never `build:`.
4. **Auto port assignment only** (`"8080"`) — never explicit mapping
   (`"8080:8080"`), even when the app normally wants a fixed host port.
5. **No Yantr labels** — do not add `yantr.app` / `yantr.service.N` /
   `yantr.port.N`. Identity comes from the app folder plus the native compose
   project/service labels. Runtime-only labels (`yantr.expireAt`,
   `yantr.temporary`, `yantr.system`) are injected by the backend.
6. **Container socket** — host side must be `${HOST_PODMAN_SOCKET}`, container
   side stays `/var/run/docker.sock`. Any other `*.sock` host source aborts the
   deploy. The placeholder is set by default on every deploy; never assume a
   fixed socket path exists.

---

## 6. Example

```yaml
# apps/my-app/compose.yml
x-yantr:
  name: "myapp"
  tags: [productivity, self-hosted, webapp, tools, podman]
  short_description: "Self-hosted note-taking app."
  description: >
    A self-hosted note-taking service that lets you capture, organize,
    and share notes with your team. Runs entirely on your own hardware.
  usecases: ["Capture notes.", "Organize docs.", "Share with team."]
  website: "https://github.com/example/my-app"
  env_generators:
    ADMIN_PASSWORD: {length: 20, charset: alnum_symbols}
  notes: ["Set admin email in the deploy form if the app requires one."]
  ports:
    - {port: 8080, protocol: HTTP, show: true, label: "Web UI", service: my-app}

services:
  my-app:
    image: ghcr.io/example/my-app:latest
    environment:
      ADMIN_USER: ${ADMIN_USER:-admin}
      ADMIN_PASSWORD: ${ADMIN_PASSWORD}
    ports:
      - "8080"
    volumes:
      - my_app_data:/data
    restart: unless-stopped

volumes:
  my_app_data:
```

---

## 7. Validation

### `check.js`

`check.js` is the fast, static app validator. It does not start Podman or pull
images: it reads the recipe and checks it against this file. Apps are visited
one at a time in sorted order, and the first hard error stops the run with the
app name, the broken rule and the compose path.

Run it from the repository root:

| Command | Scope |
| --- | --- |
| `node check.js` | every app folder, sorted |
| `node check.js --app my-app` | only `apps/my-app`; repeatable |
| `node check.js --changed` | apps touched by staged, unstaged or untracked files |
| `node check.js --since origin/main` | apps changed since a Git ref |
| `node check.js --strict` | the same scope, with warnings treated as errors |

The same modes are available as `npm run check`,
`npm run check:app -- my-app`, `npm run check:changed` and
`npm run check:strict`. Exit status is `0` when everything passed, `1` when an
app fails, and `2` for a bad flag or an unusable Git selection.

What it enforces:

- **Folder** — `compose.yml` exists and the folder contains nothing else except
  the optional `logo.svg`.
- **Recipe** — YAML parses; `x-yantr.name` is present; `tags` has 3–5 unique
  entries; `short_description` is 50–100 characters; `description` is 200–300
  characters and a folded `>` block; `usecases` has 2–5 entries; `notes` has at
  most 6; `website` is an http(s) URL.
- **Unknown keys** — any key outside the `x-yantr` vocabulary is reported, so a
  typo such as `short_desc:` or `env_generator:` does not silently disappear.
- **Developers** — every entry is a real `npub1…` key with a valid bech32
  checksum, and the same account is not listed twice.
- **Env generators** — every rule has a positive length and a known charset.
- **Ports** — each entry has a valid port, a required protocol, a label and an
  existing `service`; the same port/protocol/service is not listed twice;
  `show: true` is only allowed on `HTTP` / `HTTPS`; an app that publishes ports
  must show at least one, except the allow-listed apps with no UI (`mariadb`,
  `mongodb`, `samba`); diagnostic labels stay work ports.
- **Compose** — a service has an image and may not use `build:`; volumes are
  named (the `${HOST_PODMAN_SOCKET}` socket is the one exception); ports may not
  be fixed mappings (`"8080:80"`), they use auto assignment like `"8080"`; no
  `yantr.*` labels are written by hand.

Warnings cover rules the current catalog still misses. They do not fail a
normal run: a tag used by no other app (a one-off category or a near-synonym),
a key outside the `x-yantr` vocabulary, a `${VAR}` with no default and no
generator (for example a token the user must supply), an unused generator, an
unquoted numeric compose port, or a logo that declares no dimensions or a
non-square shape. `--strict` turns each into an error, so a new app should pass
that mode.

`check.js` is the quick fail-fast pass. The Go tests below run catalog-wide;
both must pass before an app is added.

### Go catalog tests

`core/apps/catalog_test.go` reads the real `apps/` tree and fails on:

- an app that publishes ports but marks none `show: true`;
- a `show: true` port whose protocol is not `HTTP` / `HTTPS` (a dead "Open"
  button);
- a diagnostics port (`Metrics / pprof`, `Postgres client`, `gRPC service`)
  wrongly marked `show: true`.

Databases and file shares are allow-listed as legitimately having no access
port. A new one must be added there deliberately, not by accident.

The Go tests inspect structure only (ports, developers, `build:`). Recipe text
— tags, lengths, the `>` block, unknown keys — is `check.js`'s job.

### Manual

- `notes` read well in the deploy form.
- `usecases` stay short enough to scan.
- A one-off tag is sometimes right (a genuinely new category); when it is not,
  reuse an existing spelling instead of adding a synonym (`downloads`, not
  `download`).

---

## 8. Dashboard widgets

One rule: **a widget is one or two columns wide. Never three, never four.**

```vue
<script>
// Why two at most: the widgets are shuffled once a day into a dense
// 4-column grid, so a card that takes three or four columns is the whole
// row — and where it lands reshuffles every other card from one day to the
// next. Two columns is the widest that still reads as a card.
export const meta = { cols: 2 };
</script>
```

`Home.vue` clamps the value to 4 and maps it to `.w-span-N`, so nothing stops
a widget from asking for the full grid. Two things do:

- `ui/src/utils/widgetMeta.test.js` fails the build on `cols > 2`, and on a
  widget that declares no width at all (a silent widget is a widget at an
  unknown size).
- The width itself. A two-column card is **632px** at a 1440px viewport, with
  576px of content inside its padding. Design for that: if the content does
  not fit, the content changes — not the span.

The developer band was the counterexample, and the fix is the pattern to copy:
it was `cols: 4` and built as an editorial row — identity, pitch and payment
rails side by side with hairline dividers between them. Rebuilt as
`cols: 2` it stacks, dividers become top borders, and the two QR rails still
sit two across at the bottom. It reads better than the band did at any width.

Run the UI checks with any widget change:

```sh
cd ui && npm test && npm run lint && npm run build
```
