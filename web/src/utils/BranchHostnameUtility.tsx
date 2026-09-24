import { slugify } from "@/utils/SlugUtility";

const dnsLabelMaxLen = 63;

const label = (text: string): string => slugify(text).slice(0, dnsLabelMaxLen).replace(/-+$/, "");

// Mirrors deploy.BranchDeployRule.HostnameLabel: the part a wildcard matched, or the whole branch otherwise.
export const hostnameLabel = (pattern: string, branch: string): string => {
  if (pattern.endsWith("*")) {
    const prefix = pattern.slice(0, -1);
    const matched = label(branch.startsWith(prefix) ? branch.slice(prefix.length) : branch);
    if (matched) return matched;
  }
  return label(branch);
};

export const exampleBranch = (pattern: string): string =>
  pattern.endsWith("*") ? `${pattern.slice(0, -1)}security-test` : pattern;

// A row's hostname marks the branch with *, the wizard's spelling of a template's {branch}.
export const previewHostname = (hostname: string, pattern: string, branch: string): string =>
  hostname.replaceAll("*", hostnameLabel(pattern, branch));
