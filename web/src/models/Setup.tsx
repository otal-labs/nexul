import { z } from "zod";

import type { BootstrapStatus } from "@/models/User";

export interface SetupPass {
  token: string;
  expires_at: string;
}

export interface PublicAddress {
  ipv4: string;
  ipv6?: string;
}

export const SetupStages = {
  Done: "done",
  Code: "code",
  Local: "local",
  Domain: "domain",
  Handoff: "handoff",
  GitHub: "github",
} as const;

export type SetupStage = (typeof SetupStages)[keyof typeof SetupStages];

const originOf = (url: string): string => {
  try {
    return new URL(url).origin;
  } catch {
    return "";
  }
};

// GitHub's OAuth state cookie is per-origin, so the GitHub step only runs on the stored instance URL's own origin.
export const setupStage = (status: BootstrapStatus, hasPass: boolean, origin: string): SetupStage => {
  if (status.setup_open === false) return SetupStages.Done;
  if (!hasPass) return SetupStages.Code;
  if (!status.instance_url) return status.local ? SetupStages.Local : SetupStages.Domain;
  if (originOf(status.instance_url) !== origin) return SetupStages.Handoff;
  return SetupStages.GitHub;
};

export const setupStageCopy: Record<SetupStage, { step: number; title: string; subtitle: string }> = {
  done: { step: 3, title: "Already set up", subtitle: "Someone has signed in to this instance, so setup is closed." },
  code: {
    step: 1,
    title: "Enter the setup code",
    subtitle: "The code proves you installed this server. It unlocks setup in this browser for an hour.",
  },
  local: { step: 2, title: "Setting up this computer", subtitle: "Nexul runs on localhost here, so there is no domain to set up." },
  domain: {
    step: 2,
    title: "Give Nexul a domain",
    subtitle: "GitHub sign-in needs a final https address, so the domain comes first.",
  },
  handoff: { step: 2, title: "Continue on your domain", subtitle: "The rest of setup happens on the https address." },
  github: {
    step: 3,
    title: "Connect a GitHub App",
    subtitle: "Before anyone can sign in, connect the GitHub App this instance will use.",
  },
};

// The code rides in the fragment so it never reaches a server log or Cloudflare.
export const handoffLink = (instanceUrl: string, code: string | null): string => {
  const base = `${instanceUrl.replace(/\/+$/, "")}/setup`;
  return code ? `${base}#code=${encodeURIComponent(code)}` : base;
};

export const codeFromFragment = (hash: string): string => new URLSearchParams(hash.replace(/^#/, "")).get("code") ?? "";

// Every answer must be this server; a stray AAAA elsewhere would send Let's Encrypt to the wrong host.
export const resolvesHere = (addresses: string[] | undefined, address: PublicAddress): boolean => {
  const mine = [address.ipv4, address.ipv6].filter(Boolean);
  return !!addresses && addresses.length > 0 && addresses.every((a) => mine.includes(a));
};

export const SetupCodeFormSchema = z.object({
  code: z.string().trim().min(1, "Enter the setup code"),
});

export type SetupCodeFormData = z.infer<typeof SetupCodeFormSchema>;

export const OwnHttpsFormSchema = z.object({
  url: z
    .string()
    .trim()
    .url("Enter the full address, e.g. https://deploy.example.com")
    .refine((v) => v.startsWith("https://"), "Use the https:// address. GitHub sign-in needs HTTPS."),
});

export type OwnHttpsFormData = z.infer<typeof OwnHttpsFormSchema>;

export const ProxyDomainFormSchema = z.object({
  domain: z
    .string()
    .trim()
    .toLowerCase()
    .regex(/^(?=.{1,253}$)([a-z0-9-]+\.)+[a-z]{2,}$/, "Enter a domain name like deploy.example.com, without https://"),
  email: z.union([z.literal(""), z.email("Enter a valid email or leave it empty")]),
});

export type ProxyDomainFormData = z.infer<typeof ProxyDomainFormSchema>;

export interface GitHubManifestStart {
  url: string;
  manifest: Record<string, unknown>;
}
