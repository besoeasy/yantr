/** Curated default stacks for a new homelab. IDs must match apps/<id>/. */
export const HOMELAB_KITS = [
  {
    id: "essentials",
    apps: ["adguard-home", "vaultwarden", "immich", "jellyfin", "nextcloud", "uptime-kuma"],
  },
  {
    id: "media",
    apps: ["jellyfin", "qbittorrent", "prowlarr", "radarr", "sonarr", "seerr"],
  },
  {
    id: "smarthome",
    apps: ["home-assistant", "mosquitto", "zigbee2mqtt", "esphome", "frigate", "nodered"],
  },
  {
    id: "privacy",
    apps: ["tailscale", "vaultwarden", "adguard-home", "cloudflared", "searxng", "wireguard"],
  },
  {
    id: "files",
    apps: ["nextcloud", "syncthing", "immich", "paperless-ngx", "duplicati", "kopia"],
  },
  {
    id: "monitor",
    apps: ["uptime-kuma", "glances", "grafana", "beszel", "gatus", "homepage"],
  },
];

export function getHomelabKit(id) {
  return HOMELAB_KITS.find((kit) => kit.id === id) || null;
}
