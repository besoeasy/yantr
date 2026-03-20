# Dedicated Onion Address Feature - Implementation Guide

## Privacy Design: Per-App Circuit Isolation

This implementation provides **true anonymity** by giving each app its own dedicated Tor hidden service with separate circuit routing. This prevents traffic correlation attacks where adversaries could link multiple services by observing shared Tor circuits.

### Security Model

| Approach | Privacy | Correlation Risk |
|----------|---------|------------------|
| Single shared onion | Low | High - all apps share same circuit |
| **Per-app dedicated onions** | **High** | **None - isolated circuits** |
| Manual Tor config | Medium | Medium - requires expertise |

Each app gets:
- Own `HiddenServiceDir` directory
- Own `.onion` address
- Isolated Tor circuit (no circuit sharing)
- Stealth authorization for additional protection

---

## Files Added

### 1. `daemon/tor-onion.js`
Core Tor manager providing:
- Per-app hidden service directories (`/var/lib/tor/hidden_service_<projectId>`)
- Circuit-isolated onion address creation
- Hidden service lifecycle management
- On-demand onion address retrieval

### 2. `daemon/routes/tor-onion.js`
REST API endpoints:
```
GET  /api/tor/onion              - List all onion services
GET  /api/tor/onion/:projectId   - Get onion URL for specific app
GET  /api/tor/onion/:projectId/open - Redirect to onion address
POST /api/tor/onion/enable       - Enable onion for an app
POST /api/tor/onion/disable      - Disable onion for an app
POST /api/tor/reload             - Reload Tor config
```

### 3. `apps/tor-onion-service/`
Tor daemon app template with SOCKS proxy (port 9050) and Control Port (9051).

---

## Files to Modify

### `daemon/index.js`
```javascript
import torOnionRoutes from "./routes/tor-onion.js";
import { startTor, stopTor } from "./tor-onion.js";

await fastify.register(torOnionRoutes);

startTor().catch(err => log('warn', `⚠️  [TOR] ${err.message}`));

process.on('SIGTERM', () => {
  stopTor();
});
```

### `daemon/routes/stacks.js`
Add `onionServices` to stack response:
```javascript
const onionServices = projectContainers
  .filter(c => c.Labels?.['yantr.tor.enabled'] === 'true')
  .map(c => ({
    targetPort: Number(c.Labels['yantr.tor.target.port']),
    virtualPort: Number(c.Labels['yantr.tor.virtual.port']) || 80,
    serviceName: c.Labels['yantr.tor.service.name'] || c.Labels['yantr.service'],
  }));

return reply.send({
  success: true,
  stack: {
    // ... existing fields
    onionServices,
  },
});
```

### `src/views/StackView.vue`
Add "OPEN VIA ONION" button in the header/actions area:

```vue
<template>
  <!-- Add after existing action buttons -->
  <button
    v-if="stack.onionServices?.length"
    @click="openViaOnion"
    class="inline-flex items-center gap-2 px-4 py-2 rounded-lg..."
  >
    <span class="w-2 h-2 rounded-full bg-purple-500 animate-pulse"></span>
    {{ t('stackView.openViaOnion') }}
  </button>
</template>

<script setup>
// Add to StackView.vue
async function openViaOnion() {
  const svc = stack.value.onionServices[0]
  const res = await fetch(`${apiUrl.value}/api/tor/onion/${svc.projectId}/open`)
  if (res.ok) {
    const data = await res.json()
    window.open(data.onionUrl, '_blank')
  }
}
</script>
```

Also add an "ONION" tab in the network section showing:
- Current onion address with copy button
- Port configuration
- Enable/disable toggle

---

## Label Schema

```yaml
services:
  my-app:
    labels:
      yantr.tor.enabled: "true"
      yantr.tor.target.port: "8080"      # Container port
      yantr.tor.virtual.port: "80"         # Onion virtual port
      yantr.tor.service.name: "my-app"    # Service identifier
```

---

## Usage Flow

1. **Install Tor Onion Service** app from catalog
2. **Navigate** to any app's detail page
3. **Click "OPEN VIA ONION"** button in the header
4. **Tor Browser opens** the app's dedicated `.onion` address

---

## Dashboard Button Design

The main "OPEN VIA ONION" button should be:
- Prominent placement in the stack header (next to app name)
- Purple/magenta accent color (matches Tor branding)
- Pulsing indicator showing onion is configured
- Opens Tor Browser or default browser with `.onion` URL

```
┌─────────────────────────────────────────────────────────────┐
│ ← Back    │ App Name                    [OPEN VIA ONION] 🧅│
│           │ project-id                                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Benefits

1. **No Correlation Attacks**: Each app uses a separate Tor circuit
2. **Dedicated Addresses**: Each app has its own `.onion` v3 address
3. **True Anonymity**: Traffic cannot be linked between apps
4. **Simple UX**: Single click to open via onion
5. **Self-Contained**: Works without external services
