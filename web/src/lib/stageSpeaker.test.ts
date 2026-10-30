// @vitest-environment node
import { describe, expect, it } from "vitest";

import { HOLD_MS, stepLarge, wantedLarge, type StagePerson } from "@/lib/stageSpeaker";

const people: StagePerson[] = [
  { identity: "ann", camera: false },
  { identity: "bob", camera: true },
  { identity: "cy", camera: false },
];

describe("wantedLarge", () => {
  it.each([
    { name: "nobody has spoken: the first camera", speakers: [], current: undefined, want: "bob" },
    { name: "nobody has spoken and no camera: the first person", speakers: [], current: undefined, want: "ann", only: ["ann", "cy"] },
    { name: "someone speaks: they take it", speakers: ["cy"], current: "ann", want: "cy" },
    { name: "the large one keeps it while others speak too", speakers: ["cy", "ann"], current: "ann", want: "ann" },
    { name: "everyone stops: whoever spoke last stays", speakers: [], current: "cy", want: "cy" },
    { name: "the large one left: the speaker follows", speakers: ["ann"], current: "gone", want: "ann" },
  ])("$name", ({ speakers, current, want, only }) => {
    const here = only ? people.filter((p) => only.includes(p.identity)) : people;
    expect(wantedLarge(here, speakers, current)).toBe(want);
  });

  it("is undefined for an empty call", () => {
    expect(wantedLarge([], [], undefined)).toBeUndefined();
  });
});

describe("stepLarge", () => {
  const held = { identity: "ann", since: 10_000 };

  it("holds the large tile until HOLD_MS after it last changed, then hands it over", () => {
    const early = stepLarge(held, people, ["cy"], 10_500);
    expect(early.wait).toBe(HOLD_MS - 500);
    expect(early.next).toEqual({ identity: "cy", since: 10_000 + HOLD_MS });

    expect(stepLarge(held, people, ["cy"], 10_000 + HOLD_MS)).toEqual({ next: { identity: "cy", since: 10_000 + HOLD_MS }, wait: 0 });
  });

  it("changes nothing while the large one still speaks", () => {
    expect(stepLarge(held, people, ["ann"], 10_100)).toEqual({ next: held, wait: 0 });
  });

  it("hands over at once when the large one has left", () => {
    expect(stepLarge(held, people.slice(1), ["cy"], 10_100)).toEqual({ next: { identity: "cy", since: 10_100 }, wait: 0 });
  });

  it("starts from nobody with the first camera", () => {
    expect(stepLarge(null, people, [], 500)).toEqual({ next: { identity: "bob", since: 500 }, wait: 0 });
  });
});
