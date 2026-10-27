import type { Mermaid } from "mermaid";

// The library is larger than the editor, so it loads with the first diagram on screen.
let loaded: Promise<Mermaid> | null = null;
let renders = 0;

// Inside an <img> a width of 100% has nothing to fill; the drawn width the library writes into max-width gives it one.
const sizeToContent = (svg: string): string => {
  const width = /max-width:\s*([\d.]+)px/.exec(svg)?.[1];
  return width ? svg.replace('width="100%"', `width="${width}"`) : svg;
};

// Strict mode encodes HTML in labels and drops click handlers; the caller still draws the result only through an <img>.
export const renderMermaid = async (code: string, dark: boolean): Promise<string> => {
  loaded ??= import("mermaid").then((module) => module.default);
  const mermaid = await loaded;
  mermaid.initialize({
    startOnLoad: false,
    securityLevel: "strict",
    suppressErrorRendering: true,
    htmlLabels: false,
    theme: dark ? "dark" : "neutral",
  });
  renders += 1;
  const { svg } = await mermaid.render(`mermaid-diagram-${renders}`, code);
  return sizeToContent(svg);
};

export const svgDataUrl = (svg: string): string => `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
