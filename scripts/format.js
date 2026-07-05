#!/usr/bin/env node
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import YAML from 'yaml';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const APPS_DIR = path.join(__dirname, '..', 'apps');

const apps = fs.readdirSync(APPS_DIR, { withFileTypes: true })
  .filter(e => e.isDirectory())
  .map(e => e.name);

let formatted = 0;

for (const app of apps) {
  const composePath = path.join(APPS_DIR, app, 'compose.yml');
  if (!fs.existsSync(composePath)) continue;

  const content = fs.readFileSync(composePath, 'utf8');
  const doc = YAML.parseDocument(content);

  if (!doc.contents || !doc.contents.items) continue;

  // Sort top-level keys: x-* first, then services, networks, volumes, etc.
  doc.contents.items.sort((a, b) => {
    const ka = a.key.value;
    const kb = b.key.value;
    const isXa = ka.startsWith('x-');
    const isXb = kb.startsWith('x-');
    if (isXa && !isXb) return -1;
    if (!isXa && isXb) return 1;
    // Special order for standard compose keys
    const order = ['services', 'networks', 'volumes', 'secrets', 'configs'];
    const idxA = order.indexOf(ka);
    const idxB = order.indexOf(kb);
    if (idxA !== -1 && idxB !== -1) return idxA - idxB;
    if (idxA !== -1) return -1;
    if (idxB !== -1) return 1;
    return ka.localeCompare(kb);
  });

  // Add a blank line before every top-level block (except the first one)
  for (let i = 0; i < doc.contents.items.length; i++) {
    const item = doc.contents.items[i];
    item.key.spaceBefore = (i > 0);

    // Enforce flow sequences (compact arrays) for x-yantr arrays
    if (item.key.value === 'x-yantr' && item.value && item.value.items) {
      for (const yantrProp of item.value.items) {
        if (['tags', 'usecases', 'notes'].includes(yantrProp.key.value) && yantrProp.value && yantrProp.value.items) {
          yantrProp.value.flow = true;
        }
      }
    }
  }

  const newContent = doc.toString();
  if (newContent !== content) {
    fs.writeFileSync(composePath, newContent);
    formatted++;
  }
}

console.log(`✅ Formatted ${formatted} compose.yml files alphabetically with x-yantr spacing.`);
