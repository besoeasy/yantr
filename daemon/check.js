import path from 'path';
import YAML from 'yaml';
import { fileURLToPath } from 'url';
import { dirname } from 'path';
import { readFile } from 'fs/promises';

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

        // 2) Validate yantr.port label format
        const labels = service.labels;
        if (Array.isArray(labels)) {
            for (const label of labels) {
                if (typeof label !== 'string') continue;
                const idx = label.indexOf('=');
                if (idx === -1) continue;
                const key = label.slice(0, idx).trim();
                const value = label.slice(idx + 1).trim();
                if (key === 'yantr.port') {
                    try {
                        validateYantrPortFormat(value, serviceName);
                    } catch (error) {
                        // Re-throw to stop processing - port format is critical
                        throw error;
                    }
                }
            }
        } else if (labels && typeof labels === 'object') {
            const value = labels['yantr.port'];
            if (value) {
                try {
                    validateYantrPortFormat(value, serviceName);
                } catch (error) {
                    // Re-throw to stop processing - port format is critical
                    throw error;
                }
            }
        }
    }

    return [...publishedPorts].filter(p => Number.isFinite(p) && p > 0).sort((a, b) => a - b);
}

function validateYantrPortFormat(portValue, serviceName) {
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
                `Invalid yantr.port format in service "${serviceName}": "${entry}"`
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

function extractYantrLabels(service) {
    const labels = {};
    const serviceLabels = service.labels;
    
    if (Array.isArray(serviceLabels)) {
        for (const label of serviceLabels) {
            if (typeof label !== 'string') continue;
            const idx = label.indexOf('=');
            if (idx === -1) continue;
            const key = label.slice(0, idx).trim();
            const value = label.slice(idx + 1).trim();
            if (key.startsWith('yantr.')) {
                labels[key] = value;
            }
        }
    } else if (serviceLabels && typeof serviceLabels === 'object') {
        for (const [key, value] of Object.entries(serviceLabels)) {
            if (key.startsWith('yantr.')) {
                labels[key] = value;
            }
        }
    }
    
    return labels;
}

function validateYantrLabels(appName, composeObj) {
    const errors = [];
    const warnings = [];
    const services = composeObj && typeof composeObj === 'object' ? (composeObj.services || {}) : {};

    // Check if there's at least one service with yantr labels
    let hasYantrLabels = false;
    let yantrLabels = {};
    let serviceName = '';

    for (const [svcName, service] of Object.entries(services)) {
        if (!service || typeof service !== 'object') continue;
        const labels = extractYantrLabels(service);

        if (Object.keys(labels).length > 0) {
            hasYantrLabels = true;
            yantrLabels = labels;
            serviceName = svcName;
            break;
        }
    }

    if (!hasYantrLabels) {
        errors.push(`No yantr labels found in any service`);
        return { errors, warnings };
    }

    // Check environment variable format across all services
    for (const [svcName, service] of Object.entries(services)) {
        if (!service || typeof service !== 'object') continue;
        const environment = service.environment;

        if (Array.isArray(environment)) {
            // List format detected - this is the old format
            errors.push(`Service "${svcName}" uses list format for environment variables. Use key-value format instead (e.g., "VAR: \${VAR:-default}" not "- VAR=\${VAR:-default}")`);
        } else if (environment && typeof environment === 'object') {
            for (const [key, value] of Object.entries(environment)) {
                if (value === null) continue;
                const valueType = typeof value;
                if (valueType !== 'string' && valueType !== 'number') {
                    errors.push(`Service "${svcName}" environment "${key}" has invalid type (${valueType}). Use a string, number, or null.`);
                }
            }
        }
    }

    
    // 1. Check required labels
    const requiredLabels = ['yantr.name', 'yantr.category', 'yantr.description', 'yantr.website', 'yantr.logo'];
    for (const required of requiredLabels) {
        if (!yantrLabels[required]) {
            errors.push(`Missing required label: ${required}`);
        }
    }
    
    // 2. Validate yantr.name format
    if (yantrLabels['yantr.name']) {
        const name = yantrLabels['yantr.name'];
        if (name !== name.trim()) {
            errors.push(`yantr.name has leading/trailing whitespace: "${name}"`);
        }
        if (name.length === 0) {
            errors.push(`yantr.name is empty`);
        }
    }
    
    // 3. Validate yantr.logo (if present)
    if (yantrLabels['yantr.logo']) {
        const logo = yantrLabels['yantr.logo'];
        // Check if it's an IPFS CID (basic validation)
        if (logo.includes('://')) {
            warnings.push(`yantr.logo should be an IPFS CID, not a full URL: "${logo}"`);
        } else if (!/^Qm[a-zA-Z0-9]{44}$/.test(logo) && !/^baf[a-zA-Z0-9]+$/.test(logo)) {
            warnings.push(`yantr.logo doesn't appear to be a valid IPFS CID: "${logo}"`);
        }
    }
    
    // 4. Validate yantr.category
    if (yantrLabels['yantr.category']) {
        const category = yantrLabels['yantr.category'];
        const categories = category.split(',').map(c => c.trim()).filter(Boolean);
        
        if (categories.length === 0) {
            errors.push(`yantr.category is empty`);
        } else if (categories.length > 3) {
            errors.push(`yantr.category has ${categories.length} categories (max 3 allowed)`);
        }
        
        for (const cat of categories) {
            if (cat !== cat.toLowerCase()) {
                errors.push(`yantr.category must be lowercase: "${cat}"`);
            }
            if (!/^[a-z0-9-]+$/.test(cat)) {
                errors.push(`yantr.category contains invalid characters: "${cat}" (only lowercase letters, numbers, and hyphens allowed)`);
            }
        }
    }
    
    // 5. Validate yantr.description
    if (yantrLabels['yantr.description']) {
        const description = yantrLabels['yantr.description'];
        if (description.length > 120) {
            errors.push(`yantr.description is too long (${description.length} chars, max 120): "${description.substring(0, 50)}..."`);
        }
        if (description.length === 0) {
            errors.push(`yantr.description is empty`);
        }
    }
    
    // 6. Validate yantr.website
    if (yantrLabels['yantr.website']) {
        const website = yantrLabels['yantr.website'];
        if (!website.startsWith('http://') && !website.startsWith('https://')) {
            errors.push(`yantr.website must be a valid URL starting with http:// or https://: "${website}"`);
        }
    }
    
    // 7. Validate yantr.port format (if present)
    if (yantrLabels['yantr.port']) {
        try {
            validateYantrPortFormat(yantrLabels['yantr.port'], serviceName);
        } catch (error) {
            errors.push(`yantr.port format error: ${error.message}`);
        }
    }
    
    return { errors, warnings };
}

function detectUnquotedEnvValues(composeContent) {
    const warnings = [];
    const lines = composeContent.split(/\r?\n/);

    let inServices = false;
    let servicesIndent = null;
    let currentService = null;
    let currentServiceIndent = null;
    let inEnvironment = false;
    let environmentIndent = null;

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        const trimmed = line.trim();

        if (trimmed === '' || trimmed.startsWith('#')) continue;

        const indent = line.match(/^\s*/)[0].length;

        if (!inServices) {
            if (trimmed === 'services:') {
                inServices = true;
                servicesIndent = indent;
            }
            continue;
        }

        if (indent <= servicesIndent && trimmed !== 'services:') {
            inServices = false;
            currentService = null;
            inEnvironment = false;
            continue;
        }

        if (indent === servicesIndent + 2 && trimmed.endsWith(':') && !trimmed.startsWith('-')) {
            currentService = trimmed.slice(0, -1).trim();
            currentServiceIndent = indent;
            inEnvironment = false;
            continue;
        }

        if (!currentService) continue;

        if (indent <= currentServiceIndent) {
            currentService = null;
            inEnvironment = false;
            continue;
        }

        if (trimmed === 'environment:' && indent > currentServiceIndent) {
            inEnvironment = true;
            environmentIndent = indent;
            continue;
        }

        if (!inEnvironment) continue;

        if (indent <= environmentIndent) {
            inEnvironment = false;
            continue;
        }

        if (trimmed.startsWith('- ')) {
            continue;
        }

        const colonIdx = line.indexOf(':');
        if (colonIdx === -1) continue;

        const key = line.slice(0, colonIdx).trim();
        let valueRaw = line.slice(colonIdx + 1).trim();

        if (!valueRaw) continue;
        if (valueRaw.startsWith('|') || valueRaw.startsWith('>')) continue;
        if (valueRaw.startsWith('"') || valueRaw.startsWith('\'')) continue;
        if (valueRaw.startsWith('${')) continue;

        // Strip inline comments for unquoted values.
        valueRaw = valueRaw.split(/\s+#/)[0].trim();
        if (!valueRaw) continue;

        if (valueRaw.includes(' ')) {
            warnings.push(
                `Service "${currentService}" environment "${key}" has an unquoted value with spaces. Quote it to avoid YAML parsing issues (line ${i + 1}).`
            );
        }
    }

    return warnings;
}

function extractServiceNetworkNames(service) {
    if (!service || typeof service !== 'object') return [];
    const networks = service.networks;
    if (Array.isArray(networks)) {
        return networks.filter(name => typeof name === 'string');
    }
    if (networks && typeof networks === 'object') {
        return Object.keys(networks);
    }
    return [];
}

function validateYantrNetwork(composeObj) {
    const errors = [];
    const warnings = [];
    const services = composeObj && typeof composeObj === 'object' ? (composeObj.services || {}) : {};

    const usingServices = [];
    for (const [svcName, service] of Object.entries(services)) {
        const networkNames = extractServiceNetworkNames(service);
        if (networkNames.includes('yantr_network')) {
            usingServices.push(svcName);
        }
    }

    const networkConfig = composeObj && typeof composeObj === 'object'
        ? (composeObj.networks ? composeObj.networks.yantr_network : null)
        : null;

    if (usingServices.length > 0) {
        if (!networkConfig || typeof networkConfig !== 'object') {
            errors.push(`Services [${usingServices.join(', ')}] use yantr_network but networks.yantr_network is missing`);
            return { errors, warnings };
        }

        if (networkConfig.name !== 'yantr_network') {
            errors.push('networks.yantr_network.name must be "yantr_network"');
        }

        if (networkConfig.external !== true) {
            errors.push('networks.yantr_network must set external: true');
        }

        if (Object.prototype.hasOwnProperty.call(networkConfig, 'driver')) {
            errors.push('networks.yantr_network must not set driver when using external: true');
        }
    } else if (networkConfig) {
        warnings.push('networks.yantr_network is defined but no service uses it');
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
            composeContent = await readFile(composePath, 'utf-8');
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
            composeContent = await readFile(composePath, 'utf-8');
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

        const labelResult = validateYantrLabels(appName, composeObj);
        const networkResult = validateYantrNetwork(composeObj);
        const envValueWarnings = detectUnquotedEnvValues(composeContent);
        const errors = [...labelResult.errors, ...networkResult.errors];
        const warnings = [...labelResult.warnings, ...networkResult.warnings, ...envValueWarnings];
        
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
        console.log('🔍 Running Yantr app validation...\n');
        
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
