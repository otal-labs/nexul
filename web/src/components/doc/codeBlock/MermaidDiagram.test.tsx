import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { renderMermaid } from "@/lib/mermaid";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("@/lib/mermaid", async (actual) => ({ ...(await actual<typeof import("@/lib/mermaid")>()), renderMermaid: vi.fn() }));

const renderBody = (markdown: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <RichTextEditor value={markdown} onChange={() => {}} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(renderMermaid).mockReset();
});

describe("MermaidDiagram", () => {
  it("draws a mermaid code block as an image, so script in its drawing never reaches the page", async () => {
    vi.mocked(renderMermaid).mockResolvedValue('<svg xmlns="http://www.w3.org/2000/svg"><script>window.ran = true</script></svg>');
    renderBody("```mermaid\ngraph TD\n  A-->B\n```");

    const diagram = await screen.findByRole("img", { name: "Mermaid diagram" });
    expect(diagram.getAttribute("src")).toMatch(/^data:image\/svg\+xml/);
    expect(document.querySelector("script, svg script")).toBeNull();
    expect(renderMermaid).toHaveBeenCalledWith("graph TD\n  A-->B", expect.any(Boolean));
  });

  it("says why a diagram that doesn't parse can't be drawn", async () => {
    vi.mocked(renderMermaid).mockRejectedValue(new Error("Parse error on line 2:\n...A-->\n----^"));
    renderBody("```mermaid\ngraph TD\n  A-->\n```");

    expect(await screen.findByRole("alert")).toHaveTextContent("Can't draw this diagram: Parse error on line 2:");
  });

  it("draws only the mermaid block beside a code block in another language", async () => {
    vi.mocked(renderMermaid).mockResolvedValue('<svg xmlns="http://www.w3.org/2000/svg"></svg>');
    renderBody("```js\nconst a = 1;\n```\n\n```mermaid\ngraph LR\n  A-->B\n```");

    expect(await screen.findAllByRole("img", { name: "Mermaid diagram" })).toHaveLength(1);
    expect(renderMermaid).toHaveBeenCalledOnce();
    expect(renderMermaid).toHaveBeenCalledWith("graph LR\n  A-->B", expect.any(Boolean));
  });
});
