## Genral Rules

1. Avoid using Rootful Ports (e.g., 80, 443). Use high numbered ports (e.g., 8080, 8443).
2. Use environment variables for configuration wherever possible.
3. ENsure no ports conflict between different apps.
4. Use volumes for persistent data storage. never mount to host paths directly.
5. Follow best practices for security, such as using non-root users and limiting container capabilities.
6. If needed we can ask user to input environment vars for example
      - STORAGE_MAX=${STORAGE_MAX:-200GB}


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
   - Use the exposed/published port number as string
   - Support environment variables if port is configurable
   - Example: `"3001"`, `"${WHOAMI_PORT:-8081}"`

5. **yantra.description** (Required) - Brief description of the app
   - Keep it concise (under 80 characters)
   - Describe the main purpose/functionality
   - Example: `"Network-wide ad blocking via DNS sinkhole"`

6. **yantra.website** (Required) - Official documentation or website URL
   - Link to official docs, wiki, or homepage
   - Prefer documentation over marketing pages
   - Example: `"https://docs.pi-hole.net"`

7. **yantra.github** (Required) - GitHub repository URL
   - Full URL to the main project repository
   - Example: `"https://github.com/pi-hole/pi-hole"`