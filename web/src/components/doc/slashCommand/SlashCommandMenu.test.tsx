import { act, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { SuggestionProps } from "@tiptap/suggestion";
import type { EditorView } from "@tiptap/pm/view";

import {
  SlashCommandMenu,
  type SlashCommandMenuRef,
} from "@/components/doc/slashCommand/SlashCommandMenu";
import { slashCommandItems } from "@/components/doc/slashCommand/slashCommands";

const keyEvent = (key: string) => ({ event: { key } as KeyboardEvent, view: {} as EditorView, range: { from: 0, to: 0 } });

describe("SlashCommandMenu", () => {
  afterEach(() => vi.restoreAllMocks());

  it("scrolls the newly selected option into view when the arrow keys move the selection", () => {
    const scrollIntoView = vi.spyOn(Element.prototype, "scrollIntoView").mockImplementation(() => {});
    const ref = createRef<SlashCommandMenuRef>();
    render(<SlashCommandMenu ref={ref} {...({ items: slashCommandItems, command: vi.fn() } as unknown as SuggestionProps<(typeof slashCommandItems)[number]>)} />);
    scrollIntoView.mockClear();

    act(() => {
      ref.current?.onKeyDown(keyEvent("ArrowDown"));
    });

    expect(scrollIntoView).toHaveBeenCalledWith({ block: "nearest" });
    expect(scrollIntoView.mock.contexts).toEqual([screen.getAllByRole("option")[1]]);
  });
});
