import { isSettingsSection } from "@/components/settings/SettingsNav";

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
  if (isSettingsSection(section)) return `/configuration/${section}${withQuery(params)}${hash}`;
  return undefined;
};
