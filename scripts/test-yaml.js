import YAML from 'yaml';

const doc = YAML.parseDocument(`x-yantr:
  name: "Test"
services:
  app:
    image: nginx
`);

doc.contents.items.forEach((item, i) => {
  item.key.spaceBefore = true;
});

console.log(doc.toString());
