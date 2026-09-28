import { afterEach, expect, test } from 'bun:test';
import { chmodSync, mkdirSync, mkdtempSync, rmSync, symlinkSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { spawnSync } from 'node:child_process';

const script = new URL('../public/tunnel.sh', import.meta.url).pathname;
const directories: string[] = [];

interface Options {
  os?: string;
  machine?: string;
  uid?: number;
  // Package managers and tools present on the fake machine; anything else is missing.
  tools?: ('apt-get' | 'dnf' | 'brew' | 'cloudflared' | 'sudo')[];
  // A URL the fake curl cannot reach.
  unreachable?: string;
}

// A cloudflared that records every call; installing it (apt, dnf, brew, the binary) copies this into the fake PATH.
const fakeCloudflared = `#!/bin/sh
[ "$1" = --version ] && { echo "cloudflared version 2026.9.0"; exit 0; }
printf '%s\\n' "$*" >> "$TUNNEL_TEST_DIR/cloudflared"
`;

// setup builds a machine from fakes on a PATH that holds nothing else, so the host's real apt-get or brew never leaks in.
function setup(options: Options = {}) {
  const directory = mkdtempSync(join(process.cwd(), '.tunnel-test-'));
  directories.push(directory);
  const bin = join(directory, 'bin');
  mkdirSync(bin);
  for (const tool of ['mkdir', 'mktemp', 'rm', 'head', 'cat', 'tar', 'cp', 'basename']) {
    symlinkSync(Bun.which(tool)!, join(bin, tool));
  }
  const log = (name: string) => `printf '%s\\n' "$*" >> "$TUNNEL_TEST_DIR/${name}"`;
  const addCloudflared = `cp "$TUNNEL_TEST_DIR/cloudflared.fake" "$TUNNEL_TEST_BIN/cloudflared"; chmod 755 "$TUNNEL_TEST_BIN/cloudflared"`;
  const tools = options.tools ?? ['apt-get', 'sudo'];
  const executables: Record<string, string> = {
    uname: `#!/bin/sh\n[ "$1" = -s ] && echo ${options.os ?? 'Linux'} || echo ${options.machine ?? 'x86_64'}\n`,
    id: `#!/bin/sh\necho ${options.uid ?? 1000}\n`,
    curl: `#!/bin/sh\nout=""; url=""\nwhile [ $# -gt 0 ]; do case "$1" in -o) out="$2"; shift ;; -*) ;; *) url="$1" ;; esac; shift; done\n${log('urls').replace('"$*"', '"$url"')}\n[ "$url" = "${options.unreachable ?? ''}" ] && exit 7\n[ -n "$out" ] && { echo "content of $url" > "$out"; exit 0; }\necho "content of $url"\n`,
    tee: `#!/bin/sh\n${log('tee')}\ncat > /dev/null\n`,
    install: `#!/bin/sh\n${log('install')}\ncase "$*" in */cloudflared) ${addCloudflared} ;; esac\n`,
    chmod: `#!/bin/sh\n/bin/chmod "$@"\n`,
  };
  if (tools.includes('sudo')) executables.sudo = `#!/bin/sh\n${log('sudo')}\nexec "$@"\n`;
  if (tools.includes('apt-get')) executables['apt-get'] = `#!/bin/sh\n${log('apt-get')}\n[ "$1" = install ] && { ${addCloudflared}; }\nexit 0\n`;
  if (tools.includes('dnf')) executables.dnf = `#!/bin/sh\n${log('dnf')}\n${addCloudflared}\n`;
  if (tools.includes('brew')) executables.brew = `#!/bin/sh\n${log('brew')}\n${addCloudflared}\n`;
  if (tools.includes('cloudflared')) executables.cloudflared = fakeCloudflared;
  for (const [name, body] of Object.entries(executables)) {
    writeFileSync(join(bin, name), body);
    chmodSync(join(bin, name), 0o755);
  }
  writeFileSync(join(directory, 'cloudflared.fake'), fakeCloudflared);
  const env = { PATH: bin, TUNNEL_TEST_DIR: directory, TUNNEL_TEST_BIN: bin, HOME: directory };
  return { directory, env };
}

const read = async (directory: string, name: string) => {
  const file = Bun.file(join(directory, name));
  return (await file.exists()) ? file.text() : '';
};

const run = (env: Record<string, string>, args: string[]) =>
  spawnSync('/bin/sh', [script, ...args], { env, encoding: 'utf8' });

afterEach(() => {
  for (const directory of directories.splice(0)) rmSync(directory, { recursive: true, force: true });
});

test('without exactly one token it prints the usage and changes nothing', async () => {
  const { directory, env } = setup();
  const result = run(env, []);
  expect(result.status).not.toBe(0);
  expect(result.stderr).toContain('sh -s -- <token>');
  expect(await read(directory, 'urls')).toBe('');
});

test('Windows and other systems are pointed at tunnel.ps1', () => {
  const { env } = setup({ os: 'FreeBSD' });
  const result = run(env, ['tok']);
  expect(result.status).not.toBe(0);
  expect(result.stderr).toContain('tunnel.ps1');
});

test('Debian or Ubuntu installs from Cloudflare\'s signed apt repository, then runs the service with sudo', async () => {
  const { directory, env } = setup({ tools: ['apt-get', 'sudo'] });
  const result = run(env, ['eyJtoken']);
  expect(result.status).toBe(0);
  expect(await read(directory, 'urls')).toBe('https://pkg.cloudflare.com/cloudflare-main.gpg\n');
  expect(await read(directory, 'install')).toContain('/usr/share/keyrings/cloudflare-main.gpg');
  expect(await read(directory, 'tee')).toContain('/etc/apt/sources.list.d/cloudflared.list');
  expect(await read(directory, 'apt-get')).toContain('install -y -qq cloudflared');
  expect(await read(directory, 'cloudflared')).toBe('service uninstall\nservice install eyJtoken\n');
  expect(await read(directory, 'sudo')).toContain('cloudflared service install eyJtoken');
});

test('Fedora or RHEL installs from Cloudflare\'s rpm repository', async () => {
  const { directory, env } = setup({ tools: ['dnf', 'sudo'] });
  expect(run(env, ['tok']).status).toBe(0);
  expect(await read(directory, 'urls')).toBe('https://pkg.cloudflare.com/cloudflared.repo\n');
  expect(await read(directory, 'install')).toContain('/etc/yum.repos.d/cloudflared.repo');
  expect(await read(directory, 'dnf')).toContain('install -y -q cloudflared');
  expect(await read(directory, 'cloudflared')).toContain('service install tok');
});

test('without a package manager it installs the release binary for this architecture', async () => {
  const { directory, env } = setup({ machine: 'aarch64', tools: ['sudo'] });
  expect(run(env, ['tok']).status).toBe(0);
  expect(await read(directory, 'urls')).toBe(
    'https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64\n',
  );
  expect(await read(directory, 'install')).toContain('/usr/local/bin/cloudflared');
  expect(await read(directory, 'cloudflared')).toContain('service install tok');
});

test('an installed cloudflared is reused, and a re-run replaces the service with the new token', async () => {
  const { directory, env } = setup({ tools: ['apt-get', 'sudo', 'cloudflared'] });
  const result = run(env, ['new-token']);
  expect(result.status).toBe(0);
  expect(result.stdout).toContain('cloudflared is installed (cloudflared version 2026.9.0)');
  expect(await read(directory, 'apt-get')).toBe('');
  expect(await read(directory, 'cloudflared')).toBe('service uninstall\nservice install new-token\n');
});

test('a Mac installs with Homebrew and runs the service as the user, never as root', async () => {
  const { directory, env } = setup({ os: 'Darwin', machine: 'arm64', uid: 501, tools: ['brew', 'sudo'] });
  expect(run(env, ['tok']).status).toBe(0);
  expect(await read(directory, 'brew')).toBe('install cloudflared\n');
  expect(await read(directory, 'cloudflared')).toBe('service uninstall\nservice install tok\n');
  expect(await read(directory, 'sudo')).toBe('');
});

test('as root it never calls sudo', async () => {
  const { directory, env } = setup({ uid: 0, tools: ['apt-get'] });
  expect(run(env, ['tok']).status).toBe(0);
  expect(await read(directory, 'cloudflared')).toContain('service install tok');
});

test('a download that fails stops the script before anything is installed', async () => {
  const { directory, env } = setup({ unreachable: 'https://pkg.cloudflare.com/cloudflare-main.gpg' });
  const result = run(env, ['tok']);
  expect(result.status).not.toBe(0);
  expect(result.stderr).toContain("could not download Cloudflare's package key");
  expect(await read(directory, 'apt-get')).toBe('');
  expect(await read(directory, 'cloudflared')).toBe('');
});

test('without root or sudo it says what to do', () => {
  const { env } = setup({ tools: ['apt-get'] });
  const result = run(env, ['tok']);
  expect(result.status).not.toBe(0);
  expect(result.stderr).toContain('run this as root');
});
