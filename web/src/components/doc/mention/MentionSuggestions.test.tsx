import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createRef } from "react";
import { describe, expect, it, vi } from "vitest";
import type { SuggestionProps } from "@tiptap/suggestion";
import type { EditorView } from "@tiptap/pm/view";

import {
  MentionSuggestions,
  type MentionSuggestionsRef,
} from "@/components/doc/mention/MentionSuggestions";
import type { MentionSearchResult } from "@/models/Mention";

const items: MentionSearchResult[] = [
  { type: "ticket", id: "t-1", title: "Fix the bug", status_label: "In progress", can_open: true },
  { type: "doc", id: "d-9", title: "Architecture", can_open: true },
];

const keyEvent = (key: string) => ({ event: { key } as KeyboardEvent, view: {} as EditorView, range: { from: 0, to: 0 } });

const renderSuggestions = (props: Partial<SuggestionProps<MentionSearchResult>> = {}) => {
  const ref = createRef<MentionSuggestionsRef>();
  const command = vi.fn();
  render(
    <MentionSuggestions
      ref={ref}
      {...({
        items,
        query: "fix",
        loading: false,
        command,
        ...props,
      } as SuggestionProps<MentionSearchResult>)}
    />,
  );
  return { ref, command };
};

describe("MentionSuggestions", () => {
  it("renders the picker rows grouped by kind with status labels", () => {
    renderSuggestions({});
    expect(screen.getByRole("listbox", { name: "Mention suggestions" })).toBeInTheDocument();
    expect(screen.getAllByRole("option")).toHaveLength(2);
    expect(screen.getByText("In progress")).toBeInTheDocument();
  });

  it("selects an item on Enter via keyboard", () => {
    const { ref, command } = renderSuggestions({});
    ref.current?.onKeyDown(keyEvent("Enter"));
    expect(command).toHaveBeenCalledWith(items[0]);
  });

  it("moves the selection with arrow keys and selects the highlighted row", () => {
    const { ref, command } = renderSuggestions({});
    act(() => {
      ref.current?.onKeyDown(keyEvent("ArrowDown"));
    });
    act(() => {
      ref.current?.onKeyDown(keyEvent("Enter"));
    });
    expect(command).toHaveBeenCalledWith(items[1]);
  });

  it("selects a row on click", async () => {
    const user = userEvent.setup();
    const { command } = renderSuggestions({});
    await user.click(screen.getByRole("option", { name: /Fix the bug/ }));
    expect(command).toHaveBeenCalledWith(items[0]);
  });

  it("shows a loading state before items resolve", () => {
    renderSuggestions({ items: [], loading: true });
    expect(screen.getByText("Searching…")).toBeInTheDocument();
  });

  it("shows an empty state when nothing matches", () => {
    renderSuggestions({ items: [], loading: false });
    expect(screen.getByText("No matches")).toBeInTheDocument();
  });

  it("does not swallow unrelated keys", () => {
    const { ref, command } = renderSuggestions({});
    expect(ref.current?.onKeyDown(keyEvent("Tab"))).toBe(false);
    expect(command).not.toHaveBeenCalled();
  });
});
