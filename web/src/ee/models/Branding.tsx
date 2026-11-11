import { z } from "zod";

export const DEFAULT_PRODUCT_NAME = "Nexul";

export const Appearance = { Light: "light", Dark: "dark" } as const;
export type Appearance = (typeof Appearance)[keyof typeof Appearance];

// What the page paints for one mode: token values by role. Themed means they replace a person's own palette.
export interface ModeRoles {
  themed: boolean;
  roles: Record<string, string>;
}

// GET /api/branding: public, and empty of everything while no license key unlocks branding.
export interface Look {
  enabled: boolean;
  product_name: string;
  logo_url: string;
  light: ModeRoles;
  dark: ModeRoles;
  version: string;
}

export interface BrandingTheme {
  name: string;
  appearance: Appearance;
  canvas: string;
  text: string;
  text_muted?: string;
  accent: string;
  error?: string;
  adjusted?: string[];
}

export interface BrandingLogo {
  content_type: string;
  size: number;
  version: string;
}

// GET /api/instance/branding: what is saved, and whether the license key lets it show.
export interface Branding {
  unlocked: boolean;
  product_name: string;
  accent: string;
  logo?: BrandingLogo;
  light_theme?: BrandingTheme;
  dark_theme?: BrandingTheme;
}

export const BrandingFormSchema = z.object({
  product_name: z.string().trim().max(40, "At most 40 characters"),
  accent: z.string().regex(/^(#[0-9a-fA-F]{6})?$/, "Pick a color"),
});

export type BrandingFormData = z.infer<typeof BrandingFormSchema>;

export const MAX_LOGO_BYTES = 512 * 1024;

// A look with a name is a branded instance, whatever else it sets: the product's own marketing steps back.
export const isBranded = (look: Look | undefined): boolean => !!look?.enabled && look.product_name !== "";

export const productName = (look: Look | undefined): string => (look?.enabled && look.product_name) || DEFAULT_PRODUCT_NAME;

export const themeFor = (branding: Branding, mode: Appearance): BrandingTheme | undefined =>
  mode === Appearance.Dark ? branding.dark_theme : branding.light_theme;
