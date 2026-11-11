declare const NEXUL_HOST_VERSION: string | undefined;

// The release build defines NEXUL_HOST_VERSION as its tag (build-binaries.ts); a run from source reports dev.
export const hostVersion: string = typeof NEXUL_HOST_VERSION === "string" ? NEXUL_HOST_VERSION : "dev";
