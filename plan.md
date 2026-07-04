# Yantr Daemon Rewrite Plan: Node.js to Go (`core`)

## Objective
Rewrite the existing Node.js/Fastify backend (`daemon`) in Go and rename it to `core`. This will replace the Node.js runtime with a single static binary, significantly reducing the Docker image size and improving type safety and maintainability.

## Key Motivations & Advantages
1. **Official Docker SDK**: Transition from the community `dockerode` wrapper to Docker's official Go SDK (`github.com/docker/docker/client`), providing type-safe and reliable interaction with the Docker engine.
2. **First-Party Compose Parsing**: Replace ~400 lines of custom YAML parsing and mutation in `stack-compose.js` with Docker Compose's official parser (`github.com/compose-spec/compose-go/v2`), ensuring full spec compliance.
3. **Reduced Image Size**: Eliminate the `node:alpine` runtime and `node_modules` from the Docker image, relying solely on a lightweight, static Go binary (~15-25 MB).
4. **Standard Library Capabilities**: Utilize Go's powerful standard library for critical components:
   - `net/http/httputil.ReverseProxy` for the `/browse/` dufs reverse proxy.
   - `golang.org/x/crypto/bcrypt` for secure manual bcrypt operations.
   - Standard HTTP (`net/http`) and routing (`github.com/go-chi/chi` or similar) for API endpoints.

## Proposed Module Structure

The new `core/` directory will be organized as follows:

```text
core/
├── main.go                  # Entry point, HTTP server setup, and routing
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
├── auth/
│   └── auth.go              # Stateless daku token verification and middleware
├── docker/
│   ├── client.go            # Shared docker.Client initialization
│   ├── containers.go        # Container management routes
│   ├── images.go            # Image management routes
│   ├── volumes.go           # Volume management routes
│   └── stacks.go            # Compose deploy/manage logic
├── compose/
│   └── compose.go           # Wraps compose-go: parse, transform, write operations
├── apps/
│   └── catalog.go           # Reads apps/, env_generators parsing, logo normalization
├── caddy/
│   └── caddy.go             # Caddy subprocess spawning + Caddyfile admin API
├── proxy/
│   └── browse.go            # /browse/ reverse proxy implementation
├── system/
│   └── system.go            # System status, health, version, IP identity, arch detection
└── selfinstall/
    └── selfinstall.go       # Bootstrap self-reinstall logic
```

## Action Items & Phases

### Phase 1: Scaffolding & Foundation
- [ ] Initialize the Go module (`go mod init core`).
- [ ] Set up the HTTP router and basic middleware (CORS, logging).
- [ ] Implement simple system routes: `/api/health`, `/api/version`.
- [ ] Initialize the official Docker client (`github.com/docker/docker/client`).

### Phase 2: Core Business Logic
- [ ] Implement auth verification logic (`core/auth`) matching existing daku token standards.
- [ ] Scaffold `core/apps` to parse the `apps/` directory, `info.json`, and `env_generators`.
- [ ] Integrate `compose-go` in `core/compose` for reading and transforming `compose.yml` templates.

### Phase 3: Route Porting
- [ ] Port Docker inspection routes (containers, images, volumes).
- [ ] Port the Stacks API (`/api/stacks/*`) ensuring perfect parity with the `compose-go` library for mutations.
- [ ] Implement Caddy subprocess management and API synchronization.
- [ ] Build the `/browse/` proxy using `httputil.ReverseProxy`.

### Phase 4: Self-Install & Finalization
- [ ] Port the `selfinstall` bootstrap logic for automated updates.
- [ ] Update the main `Dockerfile` to use a multi-stage Go build process.
- [ ] Perform comprehensive testing to ensure no regressions in complex compose deployments.
