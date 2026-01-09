const fs = require('fs');
const path = require('path');
const YAML = require('yaml');

function extractDefaultPortFromEnvExpression(value) {
    if (typeof value !== 'string') return null;
    const trimmed = value.trim();
    if (/^\d+$/.test(trimmed)) return Number(trimmed);

    const envDefaultMatch = trimmed.match(/\$\{[^:}]+:-([0-9]+)\}/);
    if (envDefaultMatch) return Number(envDefaultMatch[1]);

    return null;
}

function expandPortRange(value) {
    if (typeof value !== 'string') return null;
    const trimmed = value.trim();
    if (/^\d+$/.test(trimmed)) return [Number(trimmed)];

    const match = trimmed.match(/^(\d+)-(\d+)$/);
    if (!match) return null;
    const start = Number(match[1]);
    const end = Number(match[2]);
    if (!Number.isFinite(start) || !Number.isFinite(end) || start > end) return null;

    // Avoid accidentally expanding huge ranges.
    if (end - start > 1000) return null;

    const ports = [];
    for (let p = start; p <= end; p++) ports.push(p);
    return ports;
}

function extractPublishedPortsFromComposeObject(composeObj) {
    const publishedPorts = new Set();
    const services = composeObj && typeof composeObj === 'object' ? (composeObj.services || {}) : {};

    for (const service of Object.values(services)) {
        if (!service || typeof service !== 'object') continue;

        // 1) docker-compose "ports" section
        const ports = Array.isArray(service.ports) ? service.ports : [];
        for (const entry of ports) {
            if (typeof entry === 'string') {
                const noProto = entry.split('/')[0].trim();

                // Handle bracketed IPv6 like "[::1]:8080:80" by stripping the leading "[...]:"
                let portSpec = noProto;
                const bracketIdx = portSpec.lastIndexOf(']:');
                if (portSpec.startsWith('[') && bracketIdx !== -1) {
                    portSpec = portSpec.slice(bracketIdx + 2);
                }

                const parts = portSpec.split(':').map(p => p.trim()).filter(Boolean);
                // string syntax: HOST:CONTAINER or IP:HOST:CONTAINER
                let hostPart = null;
                if (parts.length === 2) hostPart = parts[0];
                if (parts.length === 3) hostPart = parts[1];
                if (!hostPart) continue;

                const maybePorts = expandPortRange(hostPart) || (extractDefaultPortFromEnvExpression(hostPart) !== null ? [extractDefaultPortFromEnvExpression(hostPart)] : null);
                if (maybePorts) {
                    for (const p of maybePorts) publishedPorts.add(p);
                }
            } else if (entry && typeof entry === 'object') {
                // object syntax: { target: 80, published: 8080, protocol: 'tcp', mode: 'host' }
                const published = entry.published;
                if (typeof published === 'number' && Number.isFinite(published)) {
                    publishedPorts.add(published);
                } else if (typeof published === 'string') {
                    const maybePorts = expandPortRange(published) || (extractDefaultPortFromEnvExpression(published) !== null ? [extractDefaultPortFromEnvExpression(published)] : null);
                    if (maybePorts) {
                        for (const p of maybePorts) publishedPorts.add(p);
                    }
                }
            }
        }

        // 2) yantra.port label (optional but useful)
        const labels = service.labels;
        if (Array.isArray(labels)) {
            for (const label of labels) {
                if (typeof label !== 'string') continue;
                const idx = label.indexOf('=');
                if (idx === -1) continue;
                const key = label.slice(0, idx).trim();
                const value = label.slice(idx + 1).trim();
                if (key !== 'yantra.port') continue;
                const port = extractDefaultPortFromEnvExpression(value);
                if (port !== null) publishedPorts.add(port);
            }
        } else if (labels && typeof labels === 'object') {
            const value = labels['yantra.port'];
            const port = typeof value === 'number' ? value : extractDefaultPortFromEnvExpression(value);
            if (port !== null && Number.isFinite(port)) publishedPorts.add(port);
        }
    }

    return [...publishedPorts].filter(p => Number.isFinite(p) && p > 0).sort((a, b) => a - b);
}

async function detectPortConflict() {
    const appsDir = path.join(__dirname, 'apps');
    const portToFiles = new Map();
    const conflictingPorts = new Set();

    let entries;
    try {
        entries = await fs.promises.readdir(appsDir, { withFileTypes: true });
    } catch {
        return { conflict: false, port: 0 };
    }

    const apps = entries.filter(e => e.isDirectory()).map(e => e.name).sort();
    for (const appName of apps) {
        const composePath = path.join(appsDir, appName, 'compose.yml');
        const composePathForOutput = path.relative(__dirname, composePath).replace(/\\/g, '/');
        let composeContent;
        try {
            composeContent = await fs.promises.readFile(composePath, 'utf8');
        } catch {
            continue;
        }

        let composeObj;
        try {
            composeObj = YAML.parse(composeContent);
        } catch {
            continue;
        }

        const ports = extractPublishedPortsFromComposeObject(composeObj);
        for (const port of ports) {
            const existingFiles = portToFiles.get(port);
            if (existingFiles) {
                existingFiles.add(composePathForOutput);
                conflictingPorts.add(port);
            } else {
                portToFiles.set(port, new Set([composePathForOutput]));
            }
        }
    }

    if (conflictingPorts.size === 0) return { conflict: false, port: 0, files: [] };

    const port = Math.min(...conflictingPorts);
    const files = [...(portToFiles.get(port) || new Set())].sort();
    return { conflict: true, port, files };
}

// Run if called directly
if (require.main === module) {
    detectPortConflict().then(result => {
        if (result.conflict) {
            console.error('❌ Port conflicts detected!');
            console.error(`Port ${result.port} is used by multiple apps:`);
            result.files.forEach(file => console.error(`  - ${file}`));
            process.exit(1);
        } else {
            console.log('✅ No port conflicts found');
            process.exit(0);
        }
    }).catch(error => {
        console.error('❌ Error running port check:', error.message);
        process.exit(1);
    });
}

module.exports = { detectPortConflict };
