// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderMermaid, svgDataUrl } from "@/lib/mermaid";

const mermaid = vi.hoisted(() => ({ initialize: vi.fn(), render: vi.fn() }));
vi.mock("mermaid", () => ({ default: mermaid }));

beforeEach(() => {
  mermaid.initialize.mockReset();
  mermaid.render.mockReset();
});

describe("renderMermaid", () => {
  it("renders in strict mode with the theme's colours and gives the drawing its own width", async () => {
    mermaid.render.mockResolvedValue({ svg: '<svg id="d" width="100%" style="max-width: 182.5px;" viewBox="0 0 182.5 90"></svg>' });

    const svg = await renderMermaid("graph TD; A-->B", false);

    expect(mermaid.initialize).toHaveBeenCalledWith(expect.objectContaining({ securityLevel: "strict", startOnLoad: false, theme: "neutral" }));
    expect(svg).toContain('width="182.5"');
    expect(svg).not.toContain('width="100%"');
  });

  it("picks the dark theme in dark mode and passes a parse error on", async () => {
    mermaid.render.mockRejectedValue(new Error("Parse error"));
    await expect(renderMermaid("graph", true)).rejects.toThrow("Parse error");
    expect(mermaid.initialize).toHaveBeenCalledWith(expect.objectContaining({ theme: "dark" }));
  });
});

describe("svgDataUrl", () => {
  it("encodes the drawing so quotes and hashes survive the URL", () => {
    expect(svgDataUrl('<svg fill="#fff"/>')).toBe("data:image/svg+xml;charset=utf-8,%3Csvg%20fill%3D%22%23fff%22%2F%3E");
  });
});
