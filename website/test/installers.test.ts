import { afterEach, expect, test } from 'bun:test';
import { createHash } from 'node:crypto';
import { chmodSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { spawnSync } from 'node:child_process';

const script = new URL('../public/install.sh', import.meta.url).pathname;
const directories: string[] = [];

// The fake nexul records its arguments and the first line it reads, then exits with installerExit.
const fakeNexul = (installerExit: number) =>
  `#!/bin/sh\nprintf '%s\\n' "$*" > "$NEXUL_TEST_DIR/args"\nread -r answer || true\nprintf '%s' "$answer" > "$NEXUL_TEST_DIR/installed"\nexit ${installerExit}\n`;

interface Options {
  os?: string;
  machine?: string;
  checksum?: string;
  installerExit?: number;
  latest?: string;
  releases?: string;
}

// setup serves a fake release through a fake curl: /releases/<tag>/<file> reads from the release directory, and
// the two GitHub API paths answer with the given JSON.
function setup(options: Options = {}) {
  const directory = mkdtempSync(join(process.cwd(), '.installer-test-'));
  directories.push(directory);
  const bin = join(directory, 'fake-bin');
  const release = join(directory, 'release');
  mkdirSync(bin);
  mkdirSync(release);
  const binary = fakeNexul(options.installerExit ?? 0);
  writeFileSync(join(release, 'nexul-linux-amd64'), binary);
  writeFileSync(join(release, 'nexul-linux-arm64'), binary);
  const sum = options.checksum ?? createHash('sha256').update(binary).digest('hex');
  writeFileSync(join(release, 'checksums.txt'), `${sum}  nexul-linux-amd64\n${sum}  nexul-linux-arm64\n`);
  writeFileSync(join(directory, 'latest.json'), options.latest ?? '{"message":"Not Found"}');
  writeFileSync(join(directory, 'releases.json'), options.releases ?? '[{"tag_name": "v0.2.0-beta.4"}]');
  const executables: Record<string, string> = {
    uname: `#!/bin/sh\n[ "$1" = -s ] && echo ${options.os ?? 'Linux'} || echo ${options.machine ?? 'x86_64'}\n`,
    id: '#!/bin/sh\necho 0\n',
    curl: `#!/bin/sh
out=""; url=""
while [ $# -gt 0 ]; do
  case "$1" in -o) out="$2"; shift ;; -*) ;; *) url="$1" ;; esac
  shift
done
printf '%s\\n' "$url" >> "$NEXUL_TEST_DIR/urls"
case "$url" in
  */releases/latest) [ -n "$(sed -n '/tag_name/p' "$NEXUL_TEST_DIR/latest.json")" ] || exit 22; src="$NEXUL_TEST_DIR/latest.json" ;;
  *'/releases?per_page=1') src="$NEXUL_TEST_DIR/releases.json" ;;
  */releases/*) src="$NEXUL_TEST_DIR/release/\${url##*/}"; printf '%s\\n' "$url" > "$NEXUL_TEST_DIR/download-url" ;;
  *) exit 22 ;;
esac
[ -f "$src" ] || exit 22
if [ -n "$out" ]; then cp "$src" "$out"; else cat "$src"; fi
`,
  };
  for (const [name, body] of Object.entries(executables)) {
    writeFileSync(join(bin, name), body);
    chmodSync(join(bin, name), 0o755);
  }
  const env = {
    ...process.env,
    PATH: `${bin}:/usr/bin:/bin`,
    NEXUL_TEST_DIR: directory,
    NEXUL_BIN_DIR: join(directory, 'installed-bin'),
    NEXUL_RELEASE_URL: 'https://example.test/releases',
    NEXUL_API_URL: 'https://api.example.test',
  };
  return { directory, env };
}

const read = (directory: string, name: string) => Bun.file(join(directory, name)).text();
const exists = (directory: string, name: string) => Bun.file(join(directory, name)).exists();

function run(env: Record<string, string | undefined>, args: string[] = []) {
  return spawnSync('/bin/sh', [script, ...args], { env, input: 'interactive input\n', encoding: 'utf8' });
}

afterEach(() => {
  for (const directory of directories.splice(0)) rmSync(directory, { recursive: true, force: true });
});

test('a non-Linux machine stops before downloading anything', async () => {
  const { directory, env } = setup({ os: 'Darwin' });
  const result = run(env);
  expect(result.status).not.toBe(0);
  expect(result.stderr).toContain('nexul serve');
  expect(await exists(directory, 'urls')).toBe(false);
});

test('an unsupported CPU stops before downloading anything', async () => {
  const { directory, env } = setup({ machine: 'riscv64' });
  expect(run(env).status).not.toBe(0);
  expect(await exists(directory, 'urls')).toBe(false);
});

test('a checksum mismatch never installs or runs the binary', async () => {
  const { directory, env } = setup({ checksum: '0'.repeat(64) });
  const result = run(env);
  expect(result.status).not.toBe(0);
  expect(result.stderr).toContain('checksum mismatch');
  expect(await exists(directory, 'installed-bin/nexul')).toBe(false);
  expect(await exists(directory, 'args')).toBe(false);
});

test('without a stable release it installs the newest beta and passes the flags through', async () => {
  const { directory, env } = setup();
  expect(run(env, ['--dir', '/srv/nexul', '--yes']).status).toBe(0);
  expect(await read(directory, 'download-url')).toBe('https://example.test/releases/v0.2.0-beta.4/checksums.txt\n');
  expect(await read(directory, 'args')).toBe('install --dir /srv/nexul --yes\n');
  expect(await read(directory, 'installed')).toBe('interactive input');
});

test('the newest stable release wins when one exists', async () => {
  const { directory, env } = setup({ latest: '{"tag_name": "v0.2.1", "name": "Nexul v0.2.1"}' });
  expect(run(env).status).toBe(0);
  expect(await read(directory, 'urls')).toContain('https://example.test/releases/v0.2.1/nexul-linux-amd64');
});

test('NEXUL_VERSION pins a release without asking the API, with or without the v', async () => {
  const { directory, env } = setup();
  expect(run({ ...env, NEXUL_VERSION: '0.1.9' }).status).toBe(0);
  const urls = await read(directory, 'urls');
  expect(urls).toContain('https://example.test/releases/v0.1.9/nexul-linux-amd64');
  expect(urls).not.toContain('api.example.test');
});

test('an arm64 machine downloads the arm64 build', async () => {
  const { directory, env } = setup({ machine: 'aarch64' });
  expect(run(env).status).toBe(0);
  expect(await read(directory, 'urls')).toContain('/nexul-linux-arm64');
});

test('piped installation reads interactive input from the controlling terminal', async () => {
  const { directory, env } = setup();
  const result = spawnSync('/usr/bin/script', ['-qefc', '/bin/cat "$NEXUL_BOOTSTRAP_TEST_SCRIPT" | /bin/sh', '/dev/null'], {
    env: { ...env, NEXUL_BOOTSTRAP_TEST_SCRIPT: script },
    input: 'interactive input\n',
    encoding: 'utf8',
  });
  expect(result.status).toBe(0);
  expect(await read(directory, 'installed')).toBe('interactive input');
});

test('installer errors propagate to the caller', () => {
  const { env } = setup({ installerExit: 31 });
  expect(run(env).status).toBe(31);
});
