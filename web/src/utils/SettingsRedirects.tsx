import { isSettingsSection, type SettingsSection } from "@/components/settings/SettingsNav";

// Sections folded into another; a bookmark to the old one lands on its replacement.
const foldedSections: Record<string, SettingsSection> = { members: "team", access: "team" };

// Where an old /configuration/<section> link lives now, or undefined if the section still exists.
export const legacyConfigurationTarget = (section: string | undefined, search: string, hash: string): string | undefined => {
  if (!section || !Object.hasOwn(foldedSections, section)) return undefined;
  return `/configuration/${foldedSections[section]}${search}${hash}`;
};

const withQuery = (params: URLSearchParams): string => (params.size > 0 ? `?${params}` : "");

// Where an old /settings/<section> link lives now (query and hash kept), or undefined if it stays in Your settings.
export const movedSettingsTarget = (section: string | undefined, search: string, hash: string): string | undefined => {
  const params = new URLSearchParams(search);

  if (section === "tokens") {
    params.set("tab", "tokens");
    return `/settings/security${withQuery(params)}${hash}`;
  }
  if (section === "instance" && params.get("tab") === "sign-in") {
    params.delete("tab");
    return `/configuration/sign-in${withQuery(params)}${hash}`;
  }
  const folded = legacyConfigurationTarget(section, withQuery(params), hash);
  if (folded) return folded;
  if (isSettingsSection(section)) return `/configuration/${section}${withQuery(params)}${hash}`;
  return undefined;
};
