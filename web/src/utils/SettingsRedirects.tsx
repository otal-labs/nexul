import { isSettingsSection } from "@/components/settings/SettingsNav";

// Where an old /settings?section=… link lives now (query and hash kept), or undefined if it stays in Your settings.
export const movedSettingsTarget = (search: string, hash: string): string | undefined => {
  const params = new URLSearchParams(search);
  const section = params.get("section");
  const tab = params.get("tab");

  if (section === "tokens") {
    params.set("section", "security");
    params.set("tab", "tokens");
    return `/settings?${params}${hash}`;
  }
  if (section === "instance" && tab === "sign-in") {
    params.set("section", "sign-in");
    params.delete("tab");
    return `/configuration?${params}${hash}`;
  }
  if (isSettingsSection(section)) return `/configuration?${params}${hash}`;
  return undefined;
};
