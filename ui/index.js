const { createApp } = Vue;

createApp({
    data() {
        return {
            activeTab: 'apps',
            containers: [],
            apps: [],
            imagesData: {},
            logsData: {},
            logFilter: 'all',
            loading: false,
            deploying: null,
            deleting: null,
            deletingImage: null,
            showEnvModal: false,
            selectedApp: null,
            envValues: {},
            appSearch: '',
            apiUrl: '',
            version: 'loading...',
            containerView: 'appstore',
            showPortModal: false,
            selectedAppForPorts: null
        }
    }, 
    computed: {
        allAppsCount() {
            // Total unique apps: installed + uninstalled only
            return this.installedApps.length + this.uninstalledApps.length;
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
                state: c.state
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
            // Combine installed apps first, then uninstalled (shuffled)
            let combined = [...this.installedApps, ...this.uninstalledApps];
            
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
            let host = window.location.hostname || 'localhost';

            // IPv6 hostnames must be wrapped in brackets when used with an explicit port
            if (host.includes(':') && !host.startsWith('[')) {
                host = `[${host}]`;
            }

            const portString = String(port ?? '').trim();
            const portMatch = portString.match(/\d+/);
            if (!portMatch) {
                return `${normalizedProtocol}://${host}`;
            }

            return `${normalizedProtocol}://${host}:${portMatch[0]}`;
        },
        getPorts(app) {
            // Parse port(s) from yantra.port label
            if (!app.port) return [];
            
            const portStr = String(app.port).trim();
            
            // Check if it's a JSON array
            if (portStr.startsWith('[')) {
                try {
                    const parsed = JSON.parse(portStr);
                    return Array.isArray(parsed) ? parsed : [portStr];
                } catch (e) {
                    return [portStr];
                }
            }
            
            // Check if it's comma-separated
            if (portStr.includes(',')) {
                return portStr.split(',').map(p => p.trim()).filter(p => p);
            }
            
            // Single port
            return [portStr];
        },
        openApp(app) {
            const ports = this.getPorts(app);
            
            if (ports.length === 0) return;
            
            if (ports.length === 1) {
                // Single port - open directly
                window.open(this.appUrl(ports[0], app.protocol || 'http'), '_blank');
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

            if (app && app.environment && app.environment.length > 0) {
                // Show modal for environment configuration
                this.selectedApp = app;
                this.envValues = {};
                // Pre-fill with defaults
                app.environment.forEach(env => {
                    this.envValues[env.envVar] = env.default;
                });
                this.showEnvModal = true;
            } else {
                // No environment variables, deploy directly
                await this.confirmDeploy(appId, {});
            }
        },
        async confirmDeploy(appId, environment) {
            this.showEnvModal = false;
            this.deploying = appId;
            try {
                const response = await fetch(`${this.apiUrl}/api/deploy`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ appId, environment })
                });
                const data = await response.json();

                if (data.success) {
                    alert(`${appId} deployed successfully!`);
                    await this.fetchContainers();
                } else {
                    alert(`Deployment failed: ${data.error}`);
                }
            } catch (error) {
                alert(`Deployment failed: ${error.message}`);
            } finally {
                this.deploying = null;
                this.selectedApp = null;
            }
        },
        cancelDeploy() {
            this.showEnvModal = false;
            this.selectedApp = null;
            this.envValues = {};
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
                    if (data.volumesFailed.length > 0) {
                        message += `\n\nWarning: Some volumes failed to remove`;
                    }
                    alert(message);
                    await this.fetchContainers();
                } else {
                    alert(`Deletion failed: ${data.error}`);
                }
            } catch (error) {
                alert(`Deletion failed: ${error.message}`);
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
                    alert(`Image deleted successfully!`);
                    await this.fetchImages();
                } else {
                    alert(`Deletion failed: ${data.error}\n${data.message}`);
                }
            } catch (error) {
                alert(`Deletion failed: ${error.message}`);
            } finally {
                this.deletingImage = null;
            }
        }
    }
}).mount('#app');
