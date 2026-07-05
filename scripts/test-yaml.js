import fs from 'fs';
import YAML from 'yaml';

const doc = YAML.parseDocument(`services:
  app:
    image: nginx
x-yantr:
  name: "Test"
volumes:
  data:
`);

// Sort top-level keys
doc.contents.items.sort((a, b) => a.key.value.localeCompare(b.key.value));

for (const item of doc.contents.items) {
  if (item.key.value === 'x-yantr') {
    item.key.spaceBefore = true; // Adds blank line before
  }
}

console.log(doc.toString());
