import { useEffect } from "react";

import { useLook } from "@/ee/hooks/BrandingHooks";
import { productName } from "@/ee/models/Branding";
import { applyThemePalette, setInstanceRoles } from "@/lib/themePalettes";
import { useThemeStore } from "@/stores/themeStore";

// The icons index.html ships, written again when the instance's own mark goes away, since the server leaves them out of a branded page.
const DEFAULT_ICONS = [
  { rel: "icon", href: "/favicon.svg", type: "image/svg+xml" },
  { rel: "icon", href: "/favicon-32.png", type: "image/png", sizes: "32x32" },
  { rel: "icon", href: "/favicon-16.png", type: "image/png", sizes: "16x16" },
  { rel: "apple-touch-icon", href: "/apple-touch-icon.png" },
];

// The instance's mark stands in for every Nexul icon; without one the page's own icons come back.
const setIcons = (logoURL: string) => {
  document.head.querySelectorAll('link[rel="icon"], link[rel="apple-touch-icon"]').forEach((link) => link.remove());
  const icons = logoURL === "" ? DEFAULT_ICONS : [{ rel: "icon", href: logoURL }];
  for (const attrs of icons) {
    const link = document.createElement("link");
    for (const [name, value] of Object.entries(attrs)) link.setAttribute(name, value);
    document.head.append(link);
  }
};

// The Owner's own rules sit last in the head, after the colours; the server already checked they load nothing.
const setCustomCSS = (css: string) => {
  const existing = document.getElementById("branding-css");
  if (css === "") return void existing?.remove();
  const style = existing ?? document.head.appendChild(Object.assign(document.createElement("style"), { id: "branding-css" }));
  style.textContent = css;
};

// Paints the instance's look onto the page: its colours, its icon and its name in the tab. Mounted once at the root, so the
// sign-in page wears the brand too, and a live change repaints every open page.
export const useApplyBranding = () => {
  const { data: look } = useLook();
  const themeId = useThemeStore((s) => s.themeId);
  const theme = useThemeStore((s) => s.theme);
  const name = productName(look);
  const logoURL = look?.enabled ? look.logo_url : "";
  const css = (look?.enabled && look.css) || "";

  useEffect(() => {
    setInstanceRoles(look?.enabled ? { light: look.light, dark: look.dark } : null);
    applyThemePalette(themeId, theme);
  }, [look, themeId, theme]);

  useEffect(() => setIcons(logoURL), [logoURL]);

  useEffect(() => setCustomCSS(css), [css]);

  useEffect(() => {
    if (!/^\(\d+\) /.test(document.title)) document.title = name;
  }, [name]);
};
