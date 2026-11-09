import {
  HTML_PAGE_COLOR_TOKENS,
  HTML_PAGE_FONT_TOKENS,
  type HtmlPageAppearance,
  type HtmlPageTheme,
} from "@nexul/client-core/htmlPage";

const FONT_FAMILIES = ["Inter Variable", "JetBrains Mono Variable", "Fraunces Variable"];

let fontFaces: string | undefined;

const absoluteUrls = (cssText: string, base: string) =>
  cssText.replace(/url\(\s*(["']?)([^"')]+)\1\s*\)/g, (match, _quote: string, url: string) => {
    if (url.startsWith("data:")) return match;
    return `url("${new URL(url, base).href}")`;
  });

const sheetFontFaces = (sheet: CSSStyleSheet): string[] => {
  try {
    const base = sheet.href ?? document.baseURI;
    return [...sheet.cssRules]
      .filter((rule): rule is CSSFontFaceRule => rule instanceof CSSFontFaceRule)
      .filter((rule) => FONT_FAMILIES.some((family) => rule.style.getPropertyValue("font-family").includes(family)))
      .map((rule) => absoluteUrls(rule.cssText, base));
  } catch {
    // A cross-origin sheet refuses to list its rules.
    return [];
  }
};

// The app's own @font-face rules, read once: the fonts ship with the build and never change while it runs.
const appFontFaces = (): string => {
  fontFaces ??= [...document.styleSheets].flatMap(sheetFontFaces).join("");
  return fontFaces;
};

// Resolves each token through a probe, so a palette's override or a color-mix arrives as a plain value.
export const readHtmlPageTheme = (appearance: HtmlPageAppearance): HtmlPageTheme => {
  const probe = document.createElement("span");
  probe.hidden = true;
  document.body.append(probe);
  const computed = getComputedStyle(probe);
  const variables: Record<string, string> = {};
  for (const token of HTML_PAGE_COLOR_TOKENS) {
    probe.style.color = `var(--${token})`;
    variables[token] = computed.color;
  }
  for (const token of HTML_PAGE_FONT_TOKENS) {
    probe.style.fontFamily = `var(--${token})`;
    variables[token] = computed.fontFamily;
  }
  probe.style.borderRadius = "var(--radius-lg)";
  variables.radius = computed.borderTopLeftRadius;
  probe.remove();
  return { appearance, variables, fontFaces: appFontFaces() };
};
