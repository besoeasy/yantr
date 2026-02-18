# What is APP ?

A app is docker container compose.yml that discribes a app.

## App Structure

/apps/app-name/compose.yml

where app-name is the application name, rules for names are

1. always smallcase
2. only aplhabets, numbers and "-" is allowed

## Compose.yml Rules

1. **Storage Guidelines**

   - Use named Docker volumes exclusively
   - **Never** mount directly to host filesystem paths
   - This ensures portability and proper isolation

2. **Environment Management**

   - Use environment variables with customisable values: `${VAR_NAME:-default_value}`
   - Avoid asking user for too much customisation, focus on environment vars which are reuired

3. **PORTS MAPPING**
   - AVOID MApping PORT OURSELVES like "HOST-PORT:CONATINER-PORT", just "CONATINER-PORT" is enough.

4. **Dependencies & Networking**

   - Each app must be deployable on its own.
   - For apps that talk to each other, attach all services to the shared external network:
     ```yaml
     networks:
       yantr_network:
         name: yantr_network
         external: true
     ```
   - Resolve dependencies via service names on `yantr_network` (for example, `http://ollama:11434`).
   - Use environment variables with sensible defaults to configure dependency endpoints.
   - Do not use `depends_on` across different apps. Only use it within the same compose.yml if needed.
   - Keep `yantr.dependencies` labels for UI metadata only.
  - If a service uses `yantr_network`, the root `networks.yantr_network` must set `external: true` and `name: yantr_network`.

## Yantr labels

We use docker labels in compose.yml to tag and work with our docker conatiner.

here is the requirements

### 1. yantr.name (Required)

- **Purpose**: Human-readable display name
- **Format**: Proper capitalization with spacing
- **Examples**:
  - `"Pi-hole"`
  - `"Uptime Kuma"`
  - `"Home Assistant"`

### 2. yantr.logo (Required)

- **Purpose**: App logo for visual identification
- **Format**: IPFS CID (Content Identifier)
- **Why IPFS**: Provides reliability and prevents URL-based payloads/tampering
- **Example**: `"QmYSoiyanJ26mbB4CVZXGNEk1tfGjNaEnf3hBQyhtgA85w"`

### 3. yantr.category (Required)

- **Purpose**: Organizes apps into browsable categories
- **Format**: Lowercase only, comma-separated for multiple categories
- **Limit**: Maximum 3 categories per app
- **Best Practice**: Reuse existing categories when possible; create new ones only when necessary
- **Examples**:
  - `"network,security"`
  - `"tools,utility"`
  - `"productivity,security,utility"`

### 4. yantr.port (Optional)

- **Purpose**: Defines primary port(s) for the app
- **When to Include**: Required if the app has a web interface
- **Format Options**:
  - Ports with labels: `"8093 (HTTPS - Web UI), 8094 (HTTPS - Admin)"`
  
### 5. yantr.description (Required)

- **Purpose**: Brief explanation of app functionality
- **Length**: Under 120 characters
- **Content**: Focus on main purpose and key features
- **Example**: `"Network-wide ad blocking via DNS sinkhole"`

### 6. yantr.website (Required)

- **Purpose**: Link to official resources
- **Preference Order**:
  1. Github repository
  2. Project homepage
