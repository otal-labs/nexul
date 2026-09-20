import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { emptyDocJson, isStructuredBody } from "@/utils/RichtextUtility";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const structured = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`;

const withMention = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"see "},{"type":"mention","attrs":{"type":"ticket","id":"t-1","label":"Fix the bug"}},{"type":"text","text":" now"}]}]}`;

const renderEditor = (
  value: string,
  onChange = () => {},
  onHeadingsChange?: (headings: { id: string; text: string; level: number }[]) => void,
) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <RichTextEditor value={value} onChange={onChange} {...(onHeadingsChange ? { onHeadingsChange } : {})} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

// jsdom has no native selection extension, so Shift+Arrow never creates a
// non-empty ProseMirror selection; Mod-a (the keymap's selectAll) does.
const selectAll = async (user: ReturnType<typeof userEvent.setup>, word: string) => {
  const body = screen.getByLabelText(/doc body/i);
  await user.click(body);
  await user.keyboard(word);
  await user.keyboard("{Control>}a{/Control}");
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.post).mockResolvedValue({ data: { chips: [] } });
});

describe("RichTextEditor", () => {
  it("renders the body as an accessible region with an empty-state hint", () => {
    renderEditor(emptyDocJson);
    expect(screen.getByLabelText(/doc body/i)).toBeInTheDocument();
    expect(screen.getByText("Start writing…")).toBeInTheDocument();
  });

  it("surfaces the formatting bubble over a text selection", async () => {
    const user = userEvent.setup();
    renderEditor(emptyDocJson);

    await selectAll(user, "Hello");

    expect(await screen.findByRole("button", { name: "Bold" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Italic" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Heading 1" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Insert link" })).toBeVisible();
  });

  it("applies a mark from the bubble and emits canonical JSON", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderEditor(emptyDocJson, onChange);

    await selectAll(user, "Hello");
    await user.click(await screen.findByRole("button", { name: "Bold" }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Bold" })).toHaveAttribute("aria-pressed", "true");
    });
    const emitted = onChange.mock.calls.at(-1)?.[0] as string;
    expect(isStructuredBody(emitted)).toBe(true);
    expect(emitted).toContain('"bold"');
    expect(emitted).toContain("Hello");
  });

  it("converts the paragraph to a heading from the bubble", async () => {
    const user = userEvent.setup();
    renderEditor(structured);

    const body = screen.getByLabelText(/doc body/i);
    await user.click(body);
    await user.keyboard("{Control>}a{/Control}");
    const h1 = await screen.findByRole("button", { name: "Heading 1" });

    // Block conversions move the selection (isActive reads the caret), so assert on
    // the produced DOM instead of aria-pressed; fireEvent clicks still work once the
    // bubble hides behind the new block.
    fireEvent.click(h1);
    await waitFor(() => {
      const heading = document.querySelector(".tiptap h1");
      expect(heading).not.toBeNull();
      expect(heading?.textContent).toBe("Hello world");
    });
  });

  it("renders existing mention nodes as live chips and resolves them in one batch", async () => {
    renderEditor(withMention);

    await waitFor(() => {
      const chip = document.querySelector('span[data-mention-type="ticket"]');
      expect(chip).not.toBeNull();
      expect(chip?.getAttribute("data-mention-id")).toBe("t-1");
    });
    expect(api.post).toHaveBeenCalledWith("/api/mentions/resolve", {
      refs: [{ type: "ticket", id: "t-1" }],
    });
  });

  it("hides the bubble once the selection is gone", async () => {
    const user = userEvent.setup();
    renderEditor(emptyDocJson);

    await selectAll(user, "Hello");
    expect(await screen.findByRole("button", { name: "Bold" })).toBeVisible();
    // jsdom has no native caret movement, but PM's keymap binds Backspace to
    // deleteSelection, which collapses the selection; the bubble plugin removes the
    // element entirely (not just hides it), so the query below resolves to null.
    await user.keyboard("{Backspace}");

    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Bold" })).not.toBeInTheDocument();
    });
  });

  it("opens the insert menu on '/' and turns the line into a heading", async () => {
    const user = userEvent.setup();
    renderEditor(emptyDocJson);

    const body = screen.getByLabelText(/doc body/i);
    await user.click(body);
    await user.keyboard("/head");

    const h1 = await screen.findByRole("option", { name: "Heading 1" });
    await user.click(h1);

    await waitFor(() => {
      const heading = document.querySelector(".tiptap h1");
      expect(heading).not.toBeNull();
      expect(heading?.textContent).toBe("");
    });
  });

  it("shows the gutter '+' only on the current empty line", async () => {
    const user = userEvent.setup();
    renderEditor(emptyDocJson);

    const body = screen.getByLabelText(/doc body/i);
    await user.click(body);
    expect(document.querySelector(".plus-menu-button")).toHaveStyle({ display: "flex" });

    await user.keyboard("Hello");
    await waitFor(() => {
      expect(document.querySelector(".plus-menu-button")).toHaveStyle({ display: "none" });
    });
  });

  it("reports the live heading outline as it's typed, not the doc's initial snapshot", async () => {
    const user = userEvent.setup();
    const onHeadingsChange = vi.fn();
    renderEditor(emptyDocJson, () => {}, onHeadingsChange);

    const body = screen.getByLabelText(/doc body/i);
    await user.click(body);
    await user.keyboard("/h1{Enter}Overview");

    await waitFor(() => {
      const last = onHeadingsChange.mock.calls.at(-1)?.[0];
      expect(last).toEqual([{ id: "overview", text: "Overview", level: 1 }]);
    });
  });
});
