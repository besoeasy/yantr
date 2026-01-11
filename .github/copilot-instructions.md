# Apps

Apps templates located in /apps/, Apps are docker apps which have templates in /apps/<app-name>/compose.yml

## General Rules

Most Import : All docker api actions should be atomic, either fully succeed or fully fail with no partial changes. beacuase our app is stateless and uses no database.

1. Run `node check.js` to check for conflicts.
2. **Configuration**: Use environment variables with defaults for configuration values (e.g., `${VAR_NAME:-default_value}`), but NOT for port mappings.
3. **Storage**: Use named volumes only. Never mount to host paths directly.
4. **File Access**: When apps need to expose folders for user file management (downloads, outputs, uploads, etc.), add a companion service using `sigoden/dufs:latest`. This provides a lightweight (~3MB) web-based file manager with upload, delete, and folder management capabilities.
   - Mount the shared volume as read-write (not `:ro`)
   - Use command: `/data --enable-cors --allow-all`
   - Expose on a separate port
   - Example: See `deluge/compose.yml` for the `deluge-http` service pattern

## Labels

All apps must include the following labels to ensure proper categorization and display in the UI:

1. **yantra.name** (Required) - Human-readable app name
   - Use proper capitalization and spacing
   - Example: `"Pi-hole"`, `"Uptime Kuma"`

2. **yantra.logo** (Required) - App logo URL

   - Should be IPFS CID
   - Prefer IPFS for reliability, since URL based images can be changed to host payloads.
   - Example: `"QmYSoiyanJ26mbB4CVZXGNEk1tfGjNaEnf3hBQyhtgA85w"`

3. **yantra.category** (Required) - App category for organization

   - Try to use already existing categories, you can create new ones if needed.
   - Use lowercase only
   - Can specify up to 3 categories as comma-separated values
   - Example: `"network,security"`, `"tools,utility"`, `"productivity,security,utility"`

4. **yantra.port** (Optional) - Primary access port for the app

   - Only required for apps with web UI
   - Can be a single port or multiple ports (comma-separated)
   - Examples:   yantra.port: "8093 (HTTPS - Web UI), 8094 (HTTPS - Admin)"
   - When multiple ports are specified, UI will show a popup to select which port to open

6. **yantra.description** (Required) - Brief description of the app

   - Keep it concise (under 80 characters)
   - Describe the main purpose/functionality
   - Example: `"Network-wide ad blocking via DNS sinkhole"`

7. **yantra.website** (Required) - Official documentation or website URL

   - Link to official docs, wiki, or homepage
   - Prefer documentation over marketing pages
   - Example: `"https://docs.pi-hole.net"`

# GitHub Copilot Instructions

After edits make sure to update index.html if we change anything.
