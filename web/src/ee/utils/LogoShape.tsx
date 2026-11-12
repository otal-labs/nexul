// A wordmark in the square mark box shrinks to a sliver; past this ratio the person is shown how it will look first.
export const OFF_SQUARE_RATIO = 1.5;

interface Size {
  width: number;
  height: number;
}

const attr = (el: Element, name: string): number => {
  const value = el.getAttribute(name) ?? "";
  return value.endsWith("%") ? 0 : Number.parseFloat(value);
};

const svgSize = (source: string): Size | null => {
  const svg = new DOMParser().parseFromString(source, "image/svg+xml").documentElement;
  const [, , vw = 0, vh = 0] = (svg.getAttribute("viewBox") ?? "").split(/[\s,]+/).map(Number);
  const width = attr(svg, "width") || vw;
  const height = attr(svg, "height") || vh;
  return width > 0 && height > 0 ? { width, height } : null;
};

/** The image's own size, or null where the browser can't say (then the logo is taken as it is). */
export const measureLogo = async (file: File): Promise<Size | null> => {
  if (file.type === "image/svg+xml") return svgSize(await file.text());
  if (typeof createImageBitmap === "undefined") return null;
  try {
    const bitmap = await createImageBitmap(file);
    const size = { width: bitmap.width, height: bitmap.height };
    bitmap.close();
    return size;
  } catch {
    return null;
  }
};

export const isOffSquare = ({ width, height }: Size): boolean => Math.max(width, height) / Math.min(width, height) > OFF_SQUARE_RATIO;
