const { createApp } = Vue;

createApp({
    data() {
        return {
            activeTab: 'containers',
            containers: [],
            apps: [],
            imagesData: {},
            logsData: {},
            logFilter: 'all',
            loading: false,
            deploying: null,
            deleting: null,
            deletingImage: null,
            deletingAllImages: false,
            showEnvModal: false,
            selectedApp: null,
            envValues: {},
            appSearch: '',
            apiUrl: '',
            version: 'loading...',
            showPortModal: false,
            selectedAppForPorts: null,
            notifications: [],
            notificationIdCounter: 0,
            musthaveapps: ["dufs","watchtower"],
            // Container detail view
            containerDetailView: false,
            selectedContainer: null,
            containerStats: null,
            containerLogs: [],
            statsInterval: null
        }
    }, 
    computed: {
        allAppsCount() {
            // Total uninstalled apps (for Apps tab badge)
            return this.uninstalledApps.length;
        },
        installedAppIds() {
            const ids = new Set(this.containers.map(c => c.app.id));
            console.log('Installed app IDs:', Array.from(ids));
            console.log('All containers:', this.containers.map(c => ({ id: c.app.id, name: c.app.name })));
            return ids;
        },
        // Custom containers are those without yantra.name label (category === 'uncategorized' and no appLabels.name)
        customContainers() {
            return this.containers.filter(c => {
                // Check if container has no yantra.name label
                return !c.appLabels || !c.appLabels.name;
            });
        },
        // Yantra-managed containers (with yantra.name label)
        yantraContainers() {
            return this.containers.filter(c => {
                return c.appLabels && c.appLabels.name;
            });
        },
        installedApps() {
            // Convert only Yantra-managed containers to app format with installed flag
            return this.yantraContainers.map(c => ({
                ...c.app,
                isInstalled: true,
                containerId: c.id,
                ports: c.ports,
                state: c.state,
                status: c.status,
                env: c.env
            }));
        },
        uninstalledApps() {
            // Filter out already installed apps and shuffle
            const uninstalled = this.apps.filter(app => {
                const isInstalled = this.installedAppIds.has(app.id);
                if (isInstalled) {
                    console.log('Filtering out installed app:', app.id, app.name);
                }
                return !isInstalled;
            });
            console.log('Uninstalled apps count:', uninstalled.length);
            return this.shuffleWithSeed(uninstalled).map(app => ({
                ...app,
                isInstalled: false
            }));
        },
        combinedApps() {
            // Only show uninstalled apps in the Apps tab
            let combined = [...this.uninstalledApps];
            
            // Apply search filter
            if (this.appSearch) {
                const search = this.appSearch.toLowerCase();
                combined = combined.filter(app => {
                    return app.name.toLowerCase().includes(search) ||
                        app.category.toLowerCase().includes(search) ||
                        (app.description && app.description.toLowerCase().includes(search));
                });
            }
            
            return combined;
        }
    }, 
    mounted() {
        this.fetchVersion();
        this.refreshAll();
        // Auto-refresh every 10 seconds
        setInterval(() => {
            if (this.activeTab === 'apps') {
                this.fetchContainers();
            }
        }, 10000);
    },
    methods: {
        // Notification system
        showNotification(message, type = 'info') {
            const id = ++this.notificationIdCounter;
            const notification = { id, message, type };
            this.notifications.push(notification);
            
            // Auto-remove after 30 seconds
            setTimeout(() => {
                this.removeNotification(id);
            }, 30000);
        },
        removeNotification(id) {
            const index = this.notifications.findIndex(n => n.id === id);
            if (index !== -1) {
                this.notifications.splice(index, 1);
            }
        },
        // SHA256 hash function
        async sha256(message) {
            const msgBuffer = new TextEncoder().encode(message);
            const hashBuffer = await crypto.subtle.digest('SHA-256', msgBuffer);
            const hashArray = Array.from(new Uint8Array(hashBuffer));
            return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        },
        
        // Get random seed based on current date-hour
        getDateHourSeed() {
            const now = new Date();
            const year = now.getFullYear();
            const month = String(now.getMonth() + 1).padStart(2, '0');
            const day = String(now.getDate()).padStart(2, '0');
            const hour = String(now.getHours()).padStart(2, '0');
            return `${year}-${month}-${day}-${hour}`;
        },
        
        // Seeded random number generator
        seededRandom(seed) {
            let x = Math.sin(seed) * 10000;
            return x - Math.floor(x);
        },
        
        // Shuffle array with seed
        shuffleWithSeed(array) {
            // Convert hash to numeric seed
            const dateHourSeed = this.getDateHourSeed();
            let numericSeed = 0;
            for (let i = 0; i < dateHourSeed.length; i++) {
                numericSeed += dateHourSeed.charCodeAt(i) * (i + 1);
            }
            
            // Clone array to avoid mutating original
            const shuffled = [...array];
            
            // Fisher-Yates shuffle with seeded random
            for (let i = shuffled.length - 1; i > 0; i--) {
                const j = Math.floor(this.seededRandom(numericSeed + i) * (i + 1));
                [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
            }
            
            return shuffled;
        },
        appUrl(port, protocol = 'http') {
            // Normalize protocol - ensure it doesn't have :// suffix
            const normalizedProtocol = protocol.replace('://', '').replace(':', '');
            console.log('[appUrl] Input - port:', port, 'protocol:', protocol, 'normalized:', normalizedProtocol);
            let host = window.location.hostname || 'localhost';

            // IPv6 hostnames must be wrapped in brackets when used with an explicit port
            if (host.includes(':') && !host.startsWith('[')) {
                host = `[${host}]`;
            }

            const portString = String(port ?? '').trim();
            const portMatch = portString.match(/\d+/);
            if (!portMatch) {
                const url = `${normalizedProtocol}://${host}`;
                console.log('[appUrl] Result (no port):', url);
                return url;
            }

            const url = `${normalizedProtocol}://${host}:${portMatch[0]}`;
            console.log('[appUrl] Result:', url);
            return url;
        },
        getPorts(app) {
            // Parse port(s) from yantra.port label
            // Supports formats:
            // - "6798 (HTTPS - Web UI)" - single port with protocol and label
            // - "6798 (HTTP - API), 6799 (HTTPS - Admin)" - multiple ports with protocols
            if (!app.port) return [];
            
            const portStr = String(app.port).trim();
            console.log('[getPorts] Input:', portStr);
            
            // Check if it's comma-separated
            if (portStr.includes(',')) {
                const result = portStr.split(',').map(p => {
                    const trimmed = p.trim();
                    // Extract port, protocol, and label: "6798 (HTTPS - Web UI)"
                    const match = trimmed.match(/^(\d+)(?:\s*\(([^-\)]+)\s*-\s*([^)]+)\))?$/);
                    if (match) {
                        return { 
                            port: match[1], 
                            protocol: match[2] ? match[2].trim().toLowerCase() : 'http',
                            label: match[3] ? match[3].trim() : null 
                        };
                    }
                    return { port: trimmed, protocol: 'http', label: null };
                }).filter(p => p.port);
                console.log('[getPorts] Multiple ports result:', result);
                return result;
            }
            
            // Single port with protocol and label
            const match = portStr.match(/^(\d+)(?:\s*\(([^-\)]+)\s*-\s*([^)]+)\))?$/);
            console.log('[getPorts] Regex match:', match);
            if (match) {
                const result = [{ 
                    port: match[1], 
                    protocol: match[2] ? match[2].trim().toLowerCase() : 'http',
                    label: match[3] ? match[3].trim() : null 
                }];
                console.log('[getPorts] Single port result:', result);
                return result;
            }
            
            return [{ port: portStr, protocol: 'http', label: null }];
        },
        openApp(app) {
            // For installed apps, use actual container ports; for uninstalled apps, use yantra.port label
            let ports;
            
            if (app.isInstalled && app.ports && app.ports.length > 0) {
                // Use actual container ports from Docker
                ports = app.ports
                    .filter(p => p.PublicPort) // Only show ports that are exposed
                    .map(p => ({
                        port: p.PublicPort,
                        protocol: p.Type === 'tcp' ? 'http' : 'http', // Default to http for now
                        label: `Port ${p.PublicPort}`
                    }));
                
                // Remove duplicates (same public port)
                const uniquePorts = [];
                const seenPorts = new Set();
                ports.forEach(p => {
                    if (!seenPorts.has(p.port)) {
                        seenPorts.add(p.port);
                        uniquePorts.push(p);
                    }
                });
                ports = uniquePorts;
            } else {
                // Use yantra.port label for uninstalled apps or as fallback
                ports = this.getPorts(app);
            }
            
            if (ports.length === 0) return;
            
            if (ports.length === 1) {
                // Single port - open directly using protocol from port
                window.open(this.appUrl(ports[0].port, ports[0].protocol), '_blank');
            } else {
                // Multiple ports - show modal
                this.selectedAppForPorts = { ...app, parsedPorts: ports };
                this.showPortModal = true;
            }
        },
        closePortModal() {
            this.showPortModal = false;
            this.selectedAppForPorts = null;
        },
        async fetchVersion() {
            try {
                const response = await fetch(`${this.apiUrl}/api/version`);
                const data = await response.json();
                this.version = data.version || '0.0.0';
            } catch (error) {
                console.error('Failed to fetch version:', error);
                this.version = '0.0.0';
            }
        },
        async refreshAll() {
            this.loading = true;
            await Promise.all([
                this.fetchContainers(),
                this.fetchApps()
            ]);
            this.loading = false;
        },
        async fetchContainers() {
            try {
                const response = await fetch(`${this.apiUrl}/api/containers`);
                const data = await response.json();
                if (data.success) {
                    this.containers = data.containers.filter(c => c.state === 'running');
                    console.log('=== CONTAINERS DATA ===');
                    console.log('Containers:', this.containers.map(c => ({ 
                        containerName: c.name, 
                        appId: c.app.id, 
                        appName: c.app.name,
                        composeProject: c.labels?.['com.docker.compose.project']
                    })));
                }
            } catch (error) {
                console.error('Failed to fetch containers:', error);
            }
        },
        async fetchApps() {
            try {
                const response = await fetch(`${this.apiUrl}/api/apps`);
                const data = await response.json();
                if (data.success) {
                    this.apps = data.apps;
                    console.log('=== APPS DATA ===');
                    console.log('Apps:', this.apps.map(a => ({ id: a.id, name: a.name })));
                }
            } catch (error) {
                console.error('Failed to fetch apps:', error);
            }
        },
        async fetchImages() {
            this.loading = true;
            try {
                const response = await fetch(`${this.apiUrl}/api/images`);
                const data = await response.json();
                if (data.success) {
                    this.imagesData = data;
                }
            } catch (error) {
                console.error('Failed to fetch images:', error);
            } finally {
                this.loading = false;
            }
        },
        async fetchLogs() {
            this.loading = true;
            try {
                const level = this.logFilter === 'all' ? '' : this.logFilter;
                const url = level ? `${this.apiUrl}/api/logs?level=${level}` : `${this.apiUrl}/api/logs`;
                const response = await fetch(url);
                const data = await response.json();
                if (data.success) {
                    this.logsData = data;
                }
            } catch (error) {
                console.error('Failed to fetch logs:', error);
            } finally {
                this.loading = false;
            }
        },
        formatTimestamp(timestamp) {
            const date = new Date(timestamp);
            return date.toLocaleTimeString('en-US', {
                hour12: false,
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        },
        async deployApp(appId) {
            // Find app to check if it has environment variables
            const app = this.apps.find(a => a.id === appId);

            if (!app) return;

            // Check if app has environment variables
            if (app.environment && app.environment.length > 0) {
                this.selectedApp = app;
                this.envValues = {};
                app.environment.forEach(env => {
                    this.envValues[env.envVar] = env.default;
                });
                this.showEnvModal = true;
            } else {
                // No environment variables, deploy directly (Docker handles ports)
                await this.confirmDeploy(appId, {}, {});
            }
        },

        async confirmDeploy(appId, environment, ports) {
            this.showEnvModal = false;
            this.showPortConfigModal = false;
            this.deploying = appId;
            
            // Show deployment started notification
            this.showNotification(`Deploying ${appId}... This may take a few minutes.`, 'info');
            
            try {
                // Create AbortController with 5 minute timeout for deployments
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), 300000); // 5 minutes
                
                const response = await fetch(`${this.apiUrl}/api/deploy`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ appId, environment }),
                    signal: controller.signal
                });

                clearTimeout(timeoutId);
                const data = await response.json();

                if (data.success) {
                    this.showNotification(`${appId} deployed successfully!`, 'success');
                    await this.fetchContainers();
                } else {
                    this.showNotification(`Deployment failed: ${data.error}`, 'error');
                }
            } catch (error) {
                if (error.name === 'AbortError') {
                    this.showNotification(`Deployment timeout - ${appId} is taking longer than expected`, 'error');
                } else {
                    this.showNotification(`Deployment failed: ${error.message}`, 'error');
                }
            } finally {
                this.deploying = null;
                this.selectedApp = null;
            }
        },
        cancelDeploy() {
            this.showEnvModal = false;
            this.showPortConfigModal = false;
            this.selectedApp = null;
            this.envValues = {};
            this.portValues = {};
        },
        async deleteContainer(containerId, containerName) {
            if (!confirm(`Delete ${containerName}?\n\nThis will remove the container and all its volumes permanently.`)) return;

            this.deleting = containerId;
            try {
                const response = await fetch(`${this.apiUrl}/api/containers/${containerId}`, {
                    method: 'DELETE'
                });
                const data = await response.json();

                if (data.success) {
                    let message = `${containerName} deleted successfully!`;
                    if (data.volumesRemoved.length > 0) {
                        message += `\n\nVolumes removed: ${data.volumesRemoved.join(', ')}`;
                    }
                    this.showNotification(message, 'success');
                    
                    // Close detail view if we're viewing this container
                    if (this.containerDetailView && this.selectedContainer?.id === containerId) {
                        this.closeContainerDetail();
                    }
                    
                    await this.fetchContainers();
                } else {
                    this.showNotification(`Deletion failed: ${data.error}`, 'error');
                }
            } catch (error) {
                this.showNotification(`Deletion failed: ${error.message}`, 'error');
            } finally {
                this.deleting = null;
            }
        },
        async deleteImage(imageId, imageName) {
            if (!confirm(`Delete image ${imageName}?\n\nThis will permanently remove the image from your system.`)) return;

            this.deletingImage = imageId;
            try {
                const response = await fetch(`${this.apiUrl}/api/images/${imageId}`, {
                    method: 'DELETE'
                });
                const data = await response.json();

                if (data.success) {
                    this.showNotification(`Image deleted successfully!`, 'success');
                    await this.fetchImages();
                } else {
                    this.showNotification(`Deletion failed: ${data.error}\n${data.message}`, 'error');
                }
            } catch (error) {
                this.showNotification(`Deletion failed: ${error.message}`, 'error');
            } finally {
                this.deletingImage = null;
            }
        },
        async deleteAllUnusedImages() {
            const count = this.imagesData.unusedImages?.length || 0;
            if (!count) return;
            
            if (!confirm(`Delete all ${count} unused images?\n\nThis will free up ${this.imagesData.unusedSize} MB of disk space.\n\nThis action cannot be undone.`)) return;

            this.deletingAllImages = true;
            let deleted = 0;
            let failed = 0;

            try {
                for (const image of this.imagesData.unusedImages) {
                    try {
                        const response = await fetch(`${this.apiUrl}/api/images/${image.id}`, {
                            method: 'DELETE'
                        });
                        const data = await response.json();
                        
                        if (data.success) {
                            deleted++;
                        } else {
                            failed++;
                        }
                    } catch (error) {
                        failed++;
                    }
                }

                if (deleted > 0) {
                    this.showNotification(`Successfully deleted ${deleted} unused image${deleted > 1 ? 's' : ''}!${failed > 0 ? `\n${failed} failed.` : ''}`, failed > 0 ? 'info' : 'success');
                    await this.fetchImages();
                } else {
                    this.showNotification(`Failed to delete images`, 'error');
                }
            } catch (error) {
                this.showNotification(`Deletion failed: ${error.message}`, 'error');
            } finally {
                this.deletingAllImages = false;
            }
        },
        // Container detail methods
        async viewContainerDetail(container) {
            this.selectedContainer = container;
            this.containerDetailView = true;
            this.containerStats = null;
            this.containerLogs = [];
            
            // Fetch initial stats and logs
            await Promise.all([
                this.fetchContainerStats(),
                this.fetchContainerLogs()
            ]);
            
            // Start polling stats every 2 seconds
            this.statsInterval = setInterval(() => {
                this.fetchContainerStats();
            }, 2000);
        },
        closeContainerDetail() {
            this.containerDetailView = false;
            this.selectedContainer = null;
            this.containerStats = null;
            this.containerLogs = [];
            
            // Stop polling stats
            if (this.statsInterval) {
                clearInterval(this.statsInterval);
                this.statsInterval = null;
            }
        },
        async fetchContainerStats() {
            if (!this.selectedContainer) return;
            
            try {
                const response = await fetch(`${this.apiUrl}/api/containers/${this.selectedContainer.id}/stats`);
                const data = await response.json();
                
                if (data.success) {
                    this.containerStats = data.stats;
                }
            } catch (error) {
                console.error('Failed to fetch container stats:', error);
            }
        },
        async fetchContainerLogs() {
            if (!this.selectedContainer) return;
            
            try {
                const response = await fetch(`${this.apiUrl}/api/containers/${this.selectedContainer.id}/logs?tail=200`);
                const data = await response.json();
                
                if (data.success) {
                    this.containerLogs = data.logs;
                }
            } catch (error) {
                console.error('Failed to fetch container logs:', error);
            }
        },
        formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        },
        getPortLabel(publicPort) {
            // Get port label from yantra.port label if available
            if (!this.selectedContainer || !this.selectedContainer.app || !this.selectedContainer.app.port) {
                return null;
            }
            
            const portStr = this.selectedContainer.app.port;
            // Parse format: "9091 (HTTP - Web Interface)"
            const regex = new RegExp(`${publicPort}\\s*\\(([^)]+)\\)`);
            const match = portStr.match(regex);
            
            if (match) {
                return match[1]; // Returns "HTTP - Web Interface"
            }
            
            return null;
        },
        getLabeledPorts() {
            // Get actual running container ports and match them with yantra.port descriptions
            if (!this.selectedContainer || !this.selectedContainer.ports || this.selectedContainer.ports.length === 0) {
                return [];
            }
            
            const ports = [];
            const portDescriptions = {};
            
            // Parse yantra.port label to build a map of descriptions
            if (this.selectedContainer.app && this.selectedContainer.app.port) {
                const portStr = this.selectedContainer.app.port;
                // Parse format: "9091 (HTTP - Web Interface), 9092 (HTTP - Downloads)"
                const regex = /(\d+)\s*\(([^-\)]+)\s*-\s*([^)]+)\)/g;
                let match;
                
                while ((match = regex.exec(portStr)) !== null) {
                    portDescriptions[match[1]] = {
                        protocol: match[2].trim().toLowerCase(),
                        label: match[3].trim()
                    };
                }
            }
            
            // Build list from actual container ports (only TCP ports with public access)
            this.selectedContainer.ports.forEach(port => {
                if (port.PublicPort && port.Type === 'tcp') {
                    // Check if we have a description for this port or any port
                    const description = portDescriptions[port.PublicPort] || portDescriptions[port.PrivatePort];
                    
                    if (description) {
                        ports.push({
                            port: port.PublicPort,
                            protocol: description.protocol,
                            label: description.label
                        });
                    } else if (Object.keys(portDescriptions).length > 0) {
                        // If we have port descriptions but none match, use first available description
                        // This handles cases where label port differs from actual port
                        const firstDesc = Object.values(portDescriptions)[ports.length];
                        if (firstDesc) {
                            ports.push({
                                port: port.PublicPort,
                                protocol: firstDesc.protocol,
                                label: firstDesc.label
                            });
                        }
                    }
                }
            });
            
            return ports;
        }
    }
}).mount('#app');
