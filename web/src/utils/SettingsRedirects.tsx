import { isSettingsSection, type SettingsSection } from "@/components/settings/SettingsNav";
import { INSTANCE_SECTION_PERMISSION, type InstanceSection } from "@/models/Access";

const isInstanceOnlySection = (section: string | undefined): section is InstanceSection =>
  !!section && Object.hasOwn(INSTANCE_SECTION_PERMISSION, section);

const withQuery = (params: URLSearchParams): string => (params.size > 0 ? `?${params}` : "");

const settingsPath = (section: SettingsSection, params: URLSearchParams, hash: string): string =>
  `/settings/${section}${withQuery(params)}${hash}`;

// Sections folded into Team, and the tab that became Sign-in providers; a bookmark to either lands on its replacement.
const foldedTarget = (section: string | undefined, params: URLSearchParams, hash: string): string | undefined => {
  if (section === "members" || section === "access") return settingsPath("team", params, hash);
  if (section !== "instance" || params.get("tab") !== "sign-in") return undefined;
  params.delete("tab");
  return settingsPath("sign-in", params, hash);
};

// Where an old /configuration/<section> link lives now (query and hash kept), or undefined if the section stayed.
// Team is not here: only a viewer holding accounts:read has it on Settings, so the page decides.
export const movedConfigurationTarget = (section: string | undefined, search: string, hash: string): string | undefined => {
  const params = new URLSearchParams(search);
  const folded = foldedTarget(section, params, hash);
  if (folded) return folded;
  if (isInstanceOnlySection(section)) return settingsPath(section, params, hash);
  return undefined;
};

// Where an old /settings/<section> link lives now (query and hash kept), or undefined if it stays on Settings.
export const movedSettingsTarget = (section: string | undefined, search: string, hash: string): string | undefined => {
  const params = new URLSearchParams(search);

  if (section === "tokens") {
    params.set("tab", "tokens");
    return `/settings/security${withQuery(params)}${hash}`;
  }
  const folded = foldedTarget(section, params, hash);
  if (folded) return folded;
  const stays = section === "team" || isInstanceOnlySection(section);
  if (!stays && isSettingsSection(section)) return `/configuration/${section}${withQuery(params)}${hash}`;
  return undefined;
};
