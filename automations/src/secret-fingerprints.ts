import { createHmac, randomBytes } from "node:crypto";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";

const KEY_BYTES = 32;

// Turns a secret value into a keyed hash, so the host can tell a value changed without keeping the value. The key never
// leaves this machine.
export class Fingerprinter {
  constructor(private readonly key: Buffer = randomBytes(KEY_BYTES)) {}

  of(value: string): string {
    return createHmac("sha256", this.key).update(value).digest("hex");
  }
}

// Reads the key from file, making it (0600) on first use. A file that cannot be read or written leaves an in-memory key,
// and `persisted` says so, so the host can log it.
export function loadFingerprinter(file: string): { fingerprinter: Fingerprinter; persisted: boolean } {
  try {
    const key = Buffer.from(readFileSync(file, "utf8").trim(), "hex");
    if (key.length === KEY_BYTES) return { fingerprinter: new Fingerprinter(key), persisted: true };
  } catch {
    // no key yet: made below
  }
  const key = randomBytes(KEY_BYTES);
  try {
    mkdirSync(dirname(file), { recursive: true, mode: 0o700 });
    writeFileSync(file, `${key.toString("hex")}\n`, { mode: 0o600 });
    return { fingerprinter: new Fingerprinter(key), persisted: true };
  } catch {
    return { fingerprinter: new Fingerprinter(key), persisted: false };
  }
}
