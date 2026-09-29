// Bumped only in the change where the app starts relying on a newer server capability.
export const MIN_SERVER_VERSION = "v0.2.0-beta.11";

interface ParsedVersion {
  release: number[];
  prerelease: (number | string)[] | null;
}

const parseVersion = (raw: string): ParsedVersion | null => {
  const match = /^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/.exec(raw.trim());
  if (!match) return null;
  const release = [Number(match[1]), Number(match[2]), Number(match[3])];
  if (match[4] === undefined) return { release, prerelease: null };
  const prerelease = match[4].split(/[.-]/).map((part) => (/^\d+$/.test(part) ? Number(part) : part));
  return { release, prerelease };
};

const comparePart = (a: number | string, b: number | string): number => {
  if (typeof a === "number" && typeof b === "number") return a - b;
  if (typeof a === "number") return -1;
  if (typeof b === "number") return 1;
  return a < b ? -1 : a > b ? 1 : 0;
};

const compareParts = (a: (number | string)[], b: (number | string)[]): number => {
  const length = Math.max(a.length, b.length);
  for (let i = 0; i < length; i += 1) {
    const left = a[i];
    const right = b[i];
    if (left === undefined) return -1;
    if (right === undefined) return 1;
    const order = comparePart(left, right);
    if (order !== 0) return order;
  }
  return 0;
};

// Semantic-version precedence: numeric pre-release parts compare as numbers and a release beats its betas.
const compareVersions = (a: string, b: string): number | null => {
  const left = parseVersion(a);
  const right = parseVersion(b);
  if (!left || !right) return null;
  const release = compareParts(left.release, right.release);
  if (release !== 0) return release;
  if (left.prerelease === null && right.prerelease === null) return 0;
  if (left.prerelease === null) return 1;
  if (right.prerelease === null) return -1;
  return compareParts(left.prerelease, right.prerelease);
};

// An unstamped local build reports "dev" and always passes; a version that cannot be parsed is refused.
export const serverIsSupported = (version: string, min = MIN_SERVER_VERSION): boolean => {
  if (version === "dev") return true;
  const order = compareVersions(version, min);
  if (order === null) return false;
  return order >= 0;
};
