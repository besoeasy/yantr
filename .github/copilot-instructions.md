# UI Design Guidelines

Ui is located in /ui/

1. Use tailwindcss & aviod custom CSS as much as possible.
2. Use VUEJS framework for all UI components.
3. Follow mobile-first design principles.
4. Use FontAwesome for icons & Avoid using image-based icons.
5. Each tailwindCSS should be mobile compatible by default, a mobile user should be able to access all features.

# Apps

Apps templates located in /apps/, Apps are docker apps which have templates in /apps/<app-name>/compose.yml

## General Rules

1. **Ports**: Avoid rootful ports (80, 443). Use high-numbered ports (5000+). Run `node check.js` to check for conflicts.
2. **Configuration**: Use environment variables with defaults (e.g., `${VAR_NAME:-default_value}`).
3. **Storage**: Use named volumes only. Never mount to host paths directly.
4. **Security**: Follow container best practices - non-root users, limited capabilities.

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
   - Can be a single port (string) or multiple ports (comma-separated)
   - Support environment variables if port is configurable
   - Examples:
     - Single port: `"3001"`, `"${WHOAMI_PORT:-8081}"`
     - Multiple ports: `"9091, 51413"`
   - When multiple ports are specified, UI will show a popup to select which port to open

5. **yantra.protocol** (Optional) - Protocol for accessing the app

   - Defaults to `"http"` if not specified
   - Can be any protocol: `"https"`, `"ftp"`, `"ssh"`, `"wss"`, etc.
   - Used when generating URLs for the "Open" button
   - Example: `"https"`, `"wss"`, `"ftp"`

6. **yantra.description** (Required) - Brief description of the app

   - Keep it concise (under 80 characters)
   - Describe the main purpose/functionality
   - Example: `"Network-wide ad blocking via DNS sinkhole"`

7. **yantra.website** (Required) - Official documentation or website URL

   - Link to official docs, wiki, or homepage
   - Prefer documentation over marketing pages
   - Example: `"https://docs.pi-hole.net"`

8. **yantra.github** (Required) - GitHub repository URL
   - Full URL to the main project repository
   - Example: `"https://github.com/pi-hole/pi-hole"`

# GitHub Copilot Instructions

After edits make sure to update index.html if we change anything.
