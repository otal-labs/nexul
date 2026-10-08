import { render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { EnterList } from "@/components/EnterList";

const rows = (ids: string[]) => ids.map((id) => <li key={id}>{id}</li>);

// jsdom has no Web Animations API; the stub records what each row was asked to play.
const animate = vi.fn();

beforeEach(() => {
  animate.mockReset();
  Element.prototype.animate = animate as unknown as Element["animate"];
});

afterEach(() => {
  delete (Element.prototype as Partial<Element>).animate;
});

const delays = () => animate.mock.calls.map(([, options]) => (options as KeyframeAnimationOptions).delay);
const flushObserver = () => new Promise((resolve) => setTimeout(resolve, 0));

describe("EnterList", () => {
  it("cascades the first eight rows 25ms apart and brings the rest in with the eighth", () => {
    render(<EnterList>{rows(["a", "b", "c", "d", "e", "f", "g", "h", "i", "j"])}</EnterList>);
    expect(delays()).toEqual([0, 25, 50, 75, 100, 125, 150, 175, 200, 200]);
  });

  it("mounts a list of fifty rows at once", () => {
    render(<EnterList>{rows(Array.from({ length: 50 }, (_, i) => `row-${i}`))}</EnterList>);
    expect(animate).not.toHaveBeenCalled();
  });

  it("keeps only the fade under reduced motion: no travel, no cascade", () => {
    vi.stubGlobal("matchMedia", (query: string) => ({ matches: query.includes("reduce") }));
    render(<EnterList>{rows(["a", "b", "c"])}</EnterList>);
    expect(animate.mock.calls.map(([keyframes]) => keyframes)).toEqual(Array(3).fill([{ opacity: 0 }, { opacity: 1 }]));
    expect(delays()).toEqual([0, 0, 0]);
    vi.unstubAllGlobals();
  });

  it("pops in a row added later but keeps a row that only moved still", async () => {
    const { rerender } = render(<EnterList>{rows(["a", "b", "c"])}</EnterList>);
    animate.mockReset();

    rerender(<EnterList>{rows(["c", "a", "b"])}</EnterList>);
    await flushObserver();
    expect(animate).not.toHaveBeenCalled();

    rerender(<EnterList>{rows(["c", "a", "b", "d"])}</EnterList>);
    await flushObserver();
    expect(animate).toHaveBeenCalledTimes(1);
    expect(animate.mock.calls[0]?.[0]).toEqual([{ opacity: 0, scale: 0.97 }, { opacity: 1, scale: 1 }]);
  });
});
