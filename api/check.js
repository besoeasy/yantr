import path from 'path';
import YAML from 'yaml';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

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

    for (const [serviceName, service] of Object.entries(services)) {
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
                
                // ONLY check ports with explicit mappings (HOST:CONTAINER format)
                // Skip auto-assigned ports (single port like "9091")
                if (parts.length < 2) {
                    // Single port - Docker will auto-assign, skip conflict check
                    continue;
                }
                
                // string syntax: HOST:CONTAINER or IP:HOST:CONTAINER
                let hostPart = null;
                if (parts.length === 2) hostPart = parts[0];
                if (parts.length === 3) hostPart = parts[1];
                if (!hostPart) continue;

                const maybePorts = expandPortRange(hostPart);
                if (maybePorts) {
                    for (const p of maybePorts) publishedPorts.add(p);
                }
            } else if (entry && typeof entry === 'object') {
                // object syntax: { target: 80, published: 8080, protocol: 'tcp', mode: 'host' }
                const published = entry.published;
                if (typeof published === 'number' && Number.isFinite(published)) {
                    publishedPorts.add(published);
                } else if (typeof published === 'string') {
                    const maybePorts = expandPortRange(published);
                    if (maybePorts) {
                        for (const p of maybePorts) publishedPorts.add(p);
                    }
                }
            }
        }

        // 2) Validate yantra.port label format
        const labels = service.labels;
        if (Array.isArray(labels)) {
            for (const label of labels) {
                if (typeof label !== 'string') continue;
                const idx = label.indexOf('=');
                if (idx === -1) continue;
                const key = label.slice(0, idx).trim();
                const value = label.slice(idx + 1).trim();
                if (key === 'yantra.port') {
                    try {
                        validateYantraPortFormat(value, serviceName);
                    } catch (error) {
                        // Re-throw to stop processing - port format is critical
                        throw error;
                    }
                }
            }
        } else if (labels && typeof labels === 'object') {
            const value = labels['yantra.port'];
            if (value) {
                try {
                    validateYantraPortFormat(value, serviceName);
                } catch (error) {
                    // Re-throw to stop processing - port format is critical
                    throw error;
                }
            }
        }
    }

    return [...publishedPorts].filter(p => Number.isFinite(p) && p > 0).sort((a, b) => a - b);
}

function validateYantraPortFormat(portValue, serviceName) {
    if (!portValue) return;
    
    const portStr = String(portValue).trim();
    
    // Valid format: "8093 (HTTPS - Web UI)" or "8093 (HTTP - API), 8094 (HTTPS - Admin)"
    // Pattern: PORT (PROTOCOL - LABEL)
    const portEntries = portStr.split(',').map(p => p.trim());
    
    for (const entry of portEntries) {
        // Must match: digits followed by (PROTOCOL - Label)
        const match = entry.match(/^(\d+)\s*\(([A-Za-z]+)\s*-\s*(.+)\)$/);
        
        if (!match) {
            throw new Error(
                `Invalid yantra.port format in service "${serviceName}": "${entry}"`
            );
        }
        
        const protocol = match[2].toLowerCase();
        const validProtocols = ['http', 'https', 'ws', 'wss', 'ftp', 'ssh', 'tcp', 'udp', 'smb'];
        
        if (!validProtocols.includes(protocol)) {
            console.warn(
                `⚠️  Warning: Uncommon protocol "${protocol}" in service "${serviceName}"\n` +
                `   Common protocols: http, https, ws, wss, ftp, ssh`
            );
        }
    }
}

function extractYantraLabels(service) {
    const labels = {};
    const serviceLabels = service.labels;
    
    if (Array.isArray(serviceLabels)) {
        for (const label of serviceLabels) {
            if (typeof label !== 'string') continue;
            const idx = label.indexOf('=');
            if (idx === -1) continue;
            const key = label.slice(0, idx).trim();
            const value = label.slice(idx + 1).trim();
            if (key.startsWith('yantra.')) {
                labels[key] = value;
            }
        }
    } else if (serviceLabels && typeof serviceLabels === 'object') {
        for (const [key, value] of Object.entries(serviceLabels)) {
            if (key.startsWith('yantra.')) {
                labels[key] = value;
            }
        }
    }
    
    return labels;
}

function validateYantraLabels(appName, composeObj) {
    const errors = [];
    const warnings = [];
    const services = composeObj && typeof composeObj === 'object' ? (composeObj.services || {}) : {};
    
    // Check if there's at least one service with yantra labels
    let hasYantraLabels = false;
    let yantraLabels = {};
    let serviceName = '';
    
    for (const [svcName, service] of Object.entries(services)) {
        if (!service || typeof service !== 'object') continue;
        const labels = extractYantraLabels(service);
        
        if (Object.keys(labels).length > 0) {
            hasYantraLabels = true;
            yantraLabels = labels;
            serviceName = svcName;
            break;
        }
    }
    
    if (!hasYantraLabels) {
        errors.push(`No yantra labels found in any service`);
        return { errors, warnings };
    }
    
    // 1. Check required labels
    const requiredLabels = ['yantra.name', 'yantra.category', 'yantra.description', 'yantra.website'];
    for (const required of requiredLabels) {
        if (!yantraLabels[required]) {
            errors.push(`Missing required label: ${required}`);
        }
    }
    
    // 2. Validate yantra.name format
    if (yantraLabels['yantra.name']) {
        const name = yantraLabels['yantra.name'];
        if (name !== name.trim()) {
            errors.push(`yantra.name has leading/trailing whitespace: "${name}"`);
        }
        if (name.length === 0) {
            errors.push(`yantra.name is empty`);
        }
    }
    
    // 3. Validate yantra.logo (if present)
    if (yantraLabels['yantra.logo']) {
        const logo = yantraLabels['yantra.logo'];
        // Check if it's an IPFS CID (basic validation)
        if (logo.includes('://')) {
            warnings.push(`yantra.logo should be an IPFS CID, not a full URL: "${logo}"`);
        } else if (!/^Qm[a-zA-Z0-9]{44}$/.test(logo) && !/^baf[a-zA-Z0-9]+$/.test(logo)) {
            warnings.push(`yantra.logo doesn't appear to be a valid IPFS CID: "${logo}"`);
        }
    }
    
    // 4. Validate yantra.category
    if (yantraLabels['yantra.category']) {
        const category = yantraLabels['yantra.category'];
        const categories = category.split(',').map(c => c.trim()).filter(Boolean);
        
        if (categories.length === 0) {
            errors.push(`yantra.category is empty`);
        } else if (categories.length > 3) {
            errors.push(`yantra.category has ${categories.length} categories (max 3 allowed)`);
        }
        
        for (const cat of categories) {
            if (cat !== cat.toLowerCase()) {
                errors.push(`yantra.category must be lowercase: "${cat}"`);
            }
            if (!/^[a-z0-9-]+$/.test(cat)) {
                errors.push(`yantra.category contains invalid characters: "${cat}" (only lowercase letters, numbers, and hyphens allowed)`);
            }
        }
    }
    
    // 5. Validate yantra.description
    if (yantraLabels['yantra.description']) {
        const description = yantraLabels['yantra.description'];
        if (description.length > 120) {
            errors.push(`yantra.description is too long (${description.length} chars, max 120): "${description.substring(0, 50)}..."`);
        }
        if (description.length === 0) {
            errors.push(`yantra.description is empty`);
        }
    }
    
    // 6. Validate yantra.website
    if (yantraLabels['yantra.website']) {
        const website = yantraLabels['yantra.website'];
        if (!website.startsWith('http://') && !website.startsWith('https://')) {
            errors.push(`yantra.website must be a valid URL starting with http:// or https://: "${website}"`);
        }
    }
    
    // 7. Validate yantra.port format (if present)
    if (yantraLabels['yantra.port']) {
        try {
            validateYantraPortFormat(yantraLabels['yantra.port'], serviceName);
        } catch (error) {
            errors.push(`yantra.port format error: ${error.message}`);
        }
    }
    
    return { errors, warnings };
}

async function detectPortConflict() {
    const appsDir = path.join(__dirname, '..', 'apps');
    const portToFiles = new Map();
    const conflictingPorts = new Set();

    let entries;
    try {
        const fs = await import('fs/promises');
        entries = await fs.readdir(appsDir, { withFileTypes: true });
    } catch {
        return { conflict: false, port: 0 };
    }

    const apps = entries.filter(e => e.isDirectory()).map(e => e.name).sort();
    for (const appName of apps) {
        const composePath = path.join(appsDir, appName, 'compose.yml');
        const composePathForOutput = path.relative(__dirname, composePath).replace(/\\/g, '/');
        let composeContent;
        try {
            composeContent = await Bun.file(composePath).text();
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

async function validateAllApps() {
    const appsDir = path.join(__dirname, '..', 'apps');
    const validationResults = [];
    let hasErrors = false;

    let entries;
    try {
        const fs = await import('fs/promises');
        entries = await fs.readdir(appsDir, { withFileTypes: true });
    } catch {
        return { hasErrors: true, results: [{ app: 'system', path: 'N/A', errors: ['Failed to read apps directory'], warnings: [] }] };
    }

    const apps = entries.filter(e => e.isDirectory()).map(e => e.name).sort();
    for (const appName of apps) {
        const composePath = path.join(appsDir, appName, 'compose.yml');
        const composePathForOutput = path.relative(__dirname, composePath).replace(/\\/g, '/');
        
        let composeContent;
        try {
            composeContent = await Bun.file(composePath).text();
        } catch {
            validationResults.push({
                app: appName,
                path: composePathForOutput,
                errors: ['compose.yml not found'],
                warnings: []
            });
            hasErrors = true;
            continue;
        }

        let composeObj;
        try {
            composeObj = YAML.parse(composeContent);
        // Validate labels (but don't stop on port format errors in this phase)
        } catch (error) {
            validationResults.push({
                app: appName,
                path: composePathForOutput,
                errors: [`Invalid YAML: ${error.message}`],
                warnings: []
            });
            hasErrors = true;
            continue;
        }

        const { errors, warnings } = validateYantraLabels(appName, composeObj);
        
        if (errors.length > 0 || warnings.length > 0) {
            validationResults.push({
                app: appName,
                path: composePathForOutput,
                errors,
                warnings
            });
            
            if (errors.length > 0) {
                hasErrors = true;
            }
        }
    }

    return { hasErrors, results: validationResults };
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
    (async () => {
        console.log('🔍 Running Yantra app validation...\n');
        
        // 1. Check port conflicts
        const portResult = await detectPortConflict();
        if (portResult.conflict) {
            console.error('❌ Port conflicts detected!');
            console.error(`Port ${portResult.port} is used by multiple apps:`);
            portResult.files.forEach(file => console.error(`  - ${file}`));
            console.error('');
        } else {
            console.log('✅ No port conflicts found');
        }
        
        // 2. Validate labels
        const labelResult = await validateAllApps();
        if (labelResult.results.length > 0) {
            console.log('\n📋 Label validation results:\n');
            
            for (const result of labelResult.results) {
                console.log(`📦 ${result.app} (${result.path})`);
                
                if (result.errors.length > 0) {
                    result.errors.forEach(error => console.error(`  ❌ ${error}`));
                }
                
                if (result.warnings.length > 0) {
                    result.warnings.forEach(warning => console.warn(`  ⚠️  ${warning}`));
                }
                
                console.log('');
            }
        } else {
            console.log('✅ All app labels are valid');
        }
        
        // Exit with error if any issues found
        if (portResult.conflict || labelResult.hasErrors) {
            process.exit(1);
        } else {
            console.log('\n✅ All checks passed!');
            process.exit(0);
        }
    })().catch(error => {
        console.error('❌ Error running validation:', error.message);
        process.exit(1);
    });
}

export { detectPortConflict, validateAllApps };
