# Yantr Codebase Refactoring Plan

This document outlines a structured, phased approach to simplifying and modularizing the Yantr codebase. By breaking down monolithic files and abstracting repetitive logic, the project will become much easier to maintain, test, and scale.

## Phase 1: Backend Monolith Decomposition (`core/main.go`)
Currently, `core/main.go` is approximately 1,800 lines long and handles routing, middleware, and all API business logic.

- **Task 1.1: Route Extraction**
  - Create new files within the `core` package (e.g., `handlers_apps.go`, `handlers_containers.go`, `handlers_stacks.go`, `handlers_volumes.go`, `handlers_system.go`).
  - Move the corresponding HTTP handler functions out of `main.go` into these domain-specific files.
  - Leave only the HTTP server initialization, Chi router setup, and middleware bindings in `main.go`.

- **Task 1.2: Docker SDK Abstraction**
  - Identify repetitive Docker client initialization and data extraction logic (e.g., container listing, label parsing, context timeouts) scattered across handlers.
  - Move this logic into helper functions inside the existing `core/docker` package.
  - *Goal*: Handlers should focus strictly on HTTP request/response handling and delegate core operations to `docker.go`.

## Phase 2: Frontend View Decomposition
Several major Vue views have grown into massive monolithic files (~700+ lines each).

- **Task 2.1: Refactor `ContainerDetail.vue`**
  - Extract the "Resources" tab into `src/components/ContainerResources.vue`.
  - Extract the "Output" (Logs) tab into `src/components/ContainerLogs.vue`.
  - Extract the "Environment Variables" tab into `src/components/ContainerEnv.vue`.

- **Task 2.2: Refactor `StackView.vue`**
  - Extract the stack service list into `src/components/StackServiceList.vue`.
  - Reuse the newly created `ContainerLogs.vue` and `ContainerResources.vue` if applicable.

- **Task 2.3: Refactor `AppDetail.vue`**
  - Extract the deployment form/configuration UI into `src/components/AppDeployForm.vue`.
  - Extract app metadata/info display into `src/components/AppMetadata.vue`.

## Phase 3: Composable & Auth Simplification
The `useYantrAuth.js` composable currently handles too many responsibilities (~330 lines).

- **Task 3.1: Isolate Cryptography**
  - Create `src/utils/crypto.js`.
  - Move the HMAC-SHA256 generation, `bufferToBase64url`, and deterministic secret generation functions here.

- **Task 3.2: Isolate Network Interception**
  - Create `src/utils/fetchInterceptor.js`.
  - Move the `window.fetch` override and URL evaluation logic (`isYantrRequest`, `shouldAttachAuth`) here.

- **Task 3.3: Refine `useYantrAuth.js`**
  - Retain only Vue reactivity (`authState`), bootstrap logic, and standard auth actions (`login`, `logout`, `setup`).

## Phase 4: Project Root Cleanup
The project root contains loose scripts that clutter the workspace.

- **Task 4.1: Script Relocation**
  - Create a `/scripts` directory at the project root.
  - Move `check.js` (validation script) and `website-build.js` into `/scripts/`.
  - Update `package.json` to reflect the new paths for `npm run website` and `node check.js`.
