import { afterEach, expect, test } from 'bun:test';
import { chmodSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { spawnSync } from 'node:child_process';

const script = new URL('../public/install.sh', import.meta.url).pathname;
const directories: string[] = [];

function setup(gitExit = 0, installerExit = 0) {
  const directory = mkdtempSync(join(process.cwd(), '.installer-test-'));
  directories.push(directory);
  const bin = join(directory, 'bin');
  mkdirSync(bin);
  const git = join(bin, 'git');
  writeFileSync(git, `#!/bin/sh
printf '%s\\n' "$*" > clone-args
[ ${gitExit} -eq 0 ] || exit ${gitExit}
/bin/mkdir nexul
printf '%s\\n' '#!/bin/sh' 'read -r answer' 'printf "%s" "$answer" > installed' 'exit ${installerExit}' > nexul/install.sh
`);
  chmodSync(git, 0o755);
  return { directory, bin };
}

function run(directory: string, bin: string) {
  return spawnSync('/bin/bash', [script], {
    cwd: directory,
    env: { ...process.env, PATH: bin },
    input: 'interactive input\n',
    encoding: 'utf8',
  });
}

function runPipedWithPty(directory: string, bin: string) {
  return spawnSync('/usr/bin/script', ['-qefc', '/bin/cat "$NEXUL_BOOTSTRAP_TEST_SCRIPT" | /bin/bash', '/dev/null'], {
    cwd: directory,
    env: { ...process.env, PATH: bin, NEXUL_BOOTSTRAP_TEST_SCRIPT: script },
    input: 'interactive input\n',
    encoding: 'utf8',
  });
}

afterEach(() => {
  for (const directory of directories.splice(0)) rmSync(directory, { recursive: true, force: true });
});

test('missing Git stops before creating an installation', async () => {
  const { directory } = setup();
  const result = run(directory, directory);
  expect(result.status).not.toBe(0);
  expect(result.stderr).toContain('Git');
  expect(await Bun.file(join(directory, 'installed')).exists()).toBe(false);
});

test('clone failure never runs an installer', async () => {
  const { directory, bin } = setup(23);
  expect(run(directory, bin).status).toBe(23);
  expect(await Bun.file(join(directory, 'installed')).exists()).toBe(false);
});

test('an existing nexul directory is left untouched', async () => {
  const { directory, bin } = setup();
  mkdirSync(join(directory, 'nexul'));
  writeFileSync(join(directory, 'nexul', 'keep'), 'existing installation');
  expect(run(directory, bin).status).not.toBe(0);
  expect(await Bun.file(join(directory, 'clone-args')).exists()).toBe(false);
  expect(await Bun.file(join(directory, 'nexul', 'keep')).text()).toBe('existing installation');
});

test('fresh installation clones the official repository and preserves interactive stdin', async () => {
  const { directory, bin } = setup();
  expect(run(directory, bin).status).toBe(0);
  expect(await Bun.file(join(directory, 'clone-args')).text()).toBe('clone https://github.com/otal-labs/nexul.git nexul\n');
  expect(await Bun.file(join(directory, 'installed')).text()).toBe('interactive input');
});

test('piped installation reads interactive input from the controlling terminal', async () => {
  const { directory, bin } = setup();
  expect(runPipedWithPty(directory, bin).status).toBe(0);
  expect(await Bun.file(join(directory, 'installed')).text()).toBe('interactive input');
});

test('installer errors propagate to the caller', () => {
  const { directory, bin } = setup(0, 31);
  expect(run(directory, bin).status).toBe(31);
});
