// An HTML page is a code block in this language: the apps run its source as a sandboxed page under the block.
export const HTML_PAGE_LANGUAGE = "html-page";

export const HTML_PAGE_MIN_HEIGHT = 40;
// Stops a page that grows with its frame, such as one sized in vh.
export const HTML_PAGE_MAX_HEIGHT = 20_000;

// The tokens a page reads as CSS variables on :root, spelled as the web app spells them.
export const HTML_PAGE_COLOR_TOKENS = [
  "background",
  "foreground",
  "card",
  "card-foreground",
  "popover",
  "popover-foreground",
  "muted",
  "muted-foreground",
  "surface-2",
  "border",
  "input",
  "focus",
  "accent",
  "primary",
  "primary-foreground",
  "brand",
  "brand-foreground",
  "success",
  "warning",
  "info",
  "merged",
  "destructive",
] as const;

export const HTML_PAGE_FONT_TOKENS = ["font-sans", "font-mono", "font-display"] as const;

export type HtmlPageAppearance = "dark" | "light";

export interface HtmlPageTheme {
  appearance: HtmlPageAppearance;
  /** Token name without its leading dashes, to a CSS value. */
  variables: Readonly<Record<string, string>>;
  /** `@font-face` rules with absolute URLs, so the page draws the app's own fonts. */
  fontFaces?: string;
}

const SOURCE = "nexul-html-page";

const BASE_CSS =
  "html{box-sizing:border-box;color:var(--foreground);font-family:var(--font-sans);font-size:14px;line-height:1.5;overflow-wrap:break-word;-webkit-font-smoothing:antialiased;-webkit-text-size-adjust:100%;scrollbar-width:none}" +
  "html::-webkit-scrollbar{display:none}body{margin:0}code,kbd,pre,samp{font-family:var(--font-mono)}" +
  // Links read as they do in the doc's own prose; :where keeps a page's own rule winning.
  ":where(a){color:var(--brand);text-underline-offset:2px}" +
  // Focus paints inside the element, so the frame's edge never clips it.
  ":where(:focus-visible){outline:2px solid var(--focus);outline-offset:-2px}";

const cleanValue = (value: string) => value.replace(/[;{}<>]/g, "");

// The theme as one stylesheet; names and values are cleaned so a value can never close the rule or the tag.
export const htmlPageThemeCss = (theme: HtmlPageTheme): string => {
  const declarations = Object.entries(theme.variables)
    .filter(([name]) => /^[a-z0-9-]+$/.test(name))
    .map(([name, value]) => `--${name}:${cleanValue(value)};`)
    .join("");
  const faces = (theme.fontFaces ?? "").replace(/<\//g, "<\\/");
  return `${faces}:root{color-scheme:${theme.appearance === "light" ? "light" : "dark"};${declarations}}${BASE_CSS}`;
};

// A page sized in vh grows with its frame forever; past this many height changes in a second it keeps the height it has.
const MAX_RESIZES_PER_SECOND = 30;

// Reports the page's height and hands web links to the host to open; a theme change rebuilds the page.
// Inside the phone's WebView the host is window.ReactNativeWebView, which takes strings.
const BOOTSTRAP_SCRIPT = `(function(){var S=${JSON.stringify(SOURCE)};function post(m){m.source=S;if(window.ReactNativeWebView){window.ReactNativeWebView.postMessage(JSON.stringify(m));return;}if(window.parent!==window)window.parent.postMessage(m,"*");}document.addEventListener("click",function(e){var l=e.composedPath().find(function(t){return t&&t.matches&&t.matches("a[href]");}),u;if(!l)return;try{u=new URL(l.getAttribute("href"),document.baseURI);}catch(x){return;}if(!/^https?:$/.test(u.protocol))return;e.preventDefault();if(e.isTrusted)post({type:"link",url:u.href});},true);var h,o,n=0,t=0;function size(){var r=document.documentElement,v=Math.ceil(r.scrollHeight>r.clientHeight?r.scrollHeight:r.getBoundingClientRect().height),now=Date.now();if(v===h)return;if(now-t>1000){t=now;n=0;}if(++n>${MAX_RESIZES_PER_SECOND}){if(o)o.disconnect();return;}h=v;post({type:"height",height:v});}if(window.ResizeObserver){o=new ResizeObserver(size);o.observe(document.documentElement);document.addEventListener("DOMContentLoaded",function(){if(document.body)o.observe(document.body);});}document.addEventListener("DOMContentLoaded",size);window.addEventListener("load",size);})();`;

const bootstrapMarkup = (scan: string, theme: HtmlPageTheme): string =>
  [
    /<meta\s[^>]*charset/i.test(scan.slice(0, 4096)) ? "" : '<meta charset="utf-8">',
    /<meta\s[^>]*name\s*=\s*["']?viewport/i.test(scan) ? "" : '<meta name="viewport" content="width=device-width, initial-scale=1">',
    // A doc's address never reaches the servers a page loads from.
    '<meta name="referrer" content="no-referrer">',
    `<style id="${SOURCE}">${htmlPageThemeCss(theme)}</style>`,
    `<script>${BOOTSTRAP_SCRIPT}</script>`,
  ].join("");

// Blanks comments and raw-text elements without shifting offsets, so a "<head>" inside a script is never the anchor.
const blankNonMarkup = (html: string): string =>
  html.replace(
    /<!--[\s\S]*?(?:-->|$)|<(script|style|textarea|title|xmp|iframe|noembed|noframes|noscript|template)\b[\s\S]*?(?:<\/\1\s*>|$)|<plaintext\b[\s\S]*$/gi,
    (match) => " ".repeat(match.length),
  );

// The block's source as a full document, with the theme and bootstrap first in its head so they run before paint.
export const htmlPageDocument = (html: string, theme: HtmlPageTheme): string => {
  const scan = blankNonMarkup(html);
  const markup = bootstrapMarkup(scan, theme);
  const headOpen = /<head(?:\s[^>]*)?>/i.exec(scan);
  if (headOpen) {
    const at = headOpen.index + headOpen[0].length;
    return html.slice(0, at) + markup + html.slice(at);
  }
  const htmlOpen = /<html(?:\s[^>]*)?>/i.exec(scan);
  if (htmlOpen) {
    const at = htmlOpen.index + htmlOpen[0].length;
    return `${html.slice(0, at)}<head>${markup}</head>${html.slice(at)}`;
  }
  const doctype = /^\s*<!doctype[^>]*>/i.exec(html);
  if (doctype) {
    const at = doctype[0].length;
    return `${html.slice(0, at)}<head>${markup}</head>${html.slice(at)}`;
  }
  return `<!doctype html><html><head>${markup}</head><body>${html}</body></html>`;
};

export type HtmlPageMessage = { type: "height"; height: number } | { type: "link"; url: string };

// Reads what a page posted; the phone's WebView delivers it as a JSON string. Anything else is ignored.
export const readHtmlPageMessage = (data: unknown): HtmlPageMessage | undefined => {
  let value = data;
  if (typeof value === "string") {
    try {
      value = JSON.parse(value);
    } catch {
      return undefined;
    }
  }
  if (typeof value !== "object" || value === null) return undefined;
  const { source, type, height, url } = value as Record<string, unknown>;
  if (source !== SOURCE) return undefined;
  if (type === "height" && typeof height === "number" && Number.isFinite(height) && height > 0) {
    return { type, height: Math.min(HTML_PAGE_MAX_HEIGHT, Math.max(HTML_PAGE_MIN_HEIGHT, Math.ceil(height))) };
  }
  if (type === "link" && typeof url === "string" && /^https?:\/\//i.test(url)) return { type, url };
  return undefined;
};
