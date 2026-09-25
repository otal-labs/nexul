import { expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const repoRoot = resolve(import.meta.dir, '../..');
const docsPath = resolve(repoRoot, 'website/src/content/docs/docs/guide/mcp-server.md');

const sortedUnique = (values: string[]) => [...new Set(values)].sort();

const productionToolNames = async (): Promise<string[]> => {
  const names: string[] = [];
  for await (const path of new Bun.Glob('internal/**/*.go').scan({ cwd: repoRoot, absolute: true })) {
    if (path.endsWith('_test.go')) continue;
    for (const match of readFileSync(path, 'utf8').matchAll(/mcptool\.New\("([a-z][a-z_]*)"/g)) {
      if (match[1]) names.push(match[1]);
    }
  }
  return sortedUnique(names);
};

const documentedToolNames = (): string[] => {
  const source = readFileSync(docsPath, 'utf8');
  const start = source.indexOf('| Area | Tools |');
  const end = source.indexOf('\n\n', start);
  expect(start).toBeGreaterThanOrEqual(0);
  expect(end).toBeGreaterThan(start);
  return sortedUnique([...source.slice(start, end).matchAll(/`([a-z][a-z_]*)`/g)].flatMap((match) => (match[1] ? [match[1]] : [])));
};

test('MCP tool table matches production registrations', async () => {
  expect(documentedToolNames()).toEqual(await productionToolNames());
});
