## Genral Rules

1. Avoid using Rootful Ports (e.g., 80, 443). Use high numbered ports (e.g., 8080, 8443).
2. Use environment variables for configuration wherever possible.
3. ENsure no ports conflict between different apps.
4. Use volumes for persistent data storage. never mount to host paths directly.
5. Follow best practices for security, such as using non-root users and limiting container capabilities.

## Labels

All apps must include the following labels to ensure proper categorization and display in the UI:

1. **yantra.name** (Required) - Human-readable app name
   - Use proper capitalization and spacing
   - Example: `"Pi-hole"`, `"Uptime Kuma"`

2. **yantra.logo** (Required) - App logo URL
   - Should be a direct image URL (PNG, SVG, or similar)
   - Prefer IPFS or CDN-hosted images for reliability
   - Example: `"https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/pi-hole.png"`

3. **yantra.category** (Required) - App category for organization
   - Valid categories: `"network"`, `"tools"`, `"productivity"`, `"utility"`, `"media"`, `"development"`, `"security"`
   - Use lowercase only
   - Example: `"network"`, `"tools"`

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

5. Follow best practices for security, such as using non-root users and limiting container capabilities.

## User Configurable Environment Variables

To ensure users can configure essential settings like Timezone and Passwords, always include the following in your `compose.yml` environment section using the standard variable substitution syntax:

```yaml
environment:
  - TZ=${TZ:-UTC}
  - WEBPASSWORD=${WEBPASSWORD:-changeme}
```

- **TZ**: Allows the user to set the container's timezone (defaults to UTC).
- **WEBPASSWORD**: Used for the application's primary admin password or web interface login (defaults to `changeme` or a safe default). **yantra.tags** (Required) - Comma-separated searchable tags
   - Use lowercase keywords related to functionality
   - Separate with commas (no spaces after commas)
   - Include 3-6 relevant tags
   - Example: `"dns,adblock,privacy,network"`
