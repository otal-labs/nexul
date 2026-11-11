import { z } from "zod";

export const LicenseState = {
  None: "none",
  Active: "active",
  Expiring: "expiring",
  Expired: "expired",
  Invalid: "invalid",
} as const;

export type LicenseState = (typeof LicenseState)[keyof typeof LicenseState];

export const LicenseSchema = z.object({
  state: z.enum([LicenseState.None, LicenseState.Active, LicenseState.Expiring, LicenseState.Expired, LicenseState.Invalid]),
  licensee: z.string(),
  instance_url: z.string(),
  features: z.array(z.string()),
  expires_at: z.string().optional(),
  problem: z.string().optional(),
});

export type License = z.infer<typeof LicenseSchema>;

export const LicenseKeyFormSchema = z.object({
  key: z.string().trim().min(1, "Paste the license key"),
});

export type LicenseKeyFormData = z.infer<typeof LicenseKeyFormSchema>;

const featureLabels: Record<string, string> = { sso: "Single sign-on", branding: "Branding" };

// A feature a newer build added shows by its name until this build learns its label.
export const featureLabel = (feature: string): string => featureLabels[feature] ?? feature;

// The Owner is told while the key runs out, once it has, and while it no longer fits this instance.
export const needsNotice = (license: License): boolean =>
  license.state === LicenseState.Expiring || license.state === LicenseState.Expired || license.state === LicenseState.Invalid;

export const formatExpiry = (iso: string): string =>
  new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(new Date(iso));
