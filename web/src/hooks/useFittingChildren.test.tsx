import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useFittingChildren } from "@/hooks/useFittingChildren";

// Lays out fake children: each is [top, left, width] inside a 100px container.
const place = (element: HTMLElement, boxes: [number, number, number][]) => {
  Object.defineProperty(element, "clientWidth", { value: 100 });
  boxes.forEach(([top, left, width], index) => {
    const child = element.children[index]!;
    Object.defineProperty(child, "offsetTop", { value: top });
    Object.defineProperty(child, "offsetLeft", { value: left });
    Object.defineProperty(child, "offsetWidth", { value: width });
  });
};

const Probe = ({ boxes }: { boxes: [number, number, number][] }) => {
  const [ref, fitting, slack] = useFittingChildren<HTMLDivElement>(boxes.length);
  return (
    <>
      <div ref={(element) => {
        if (element) place(element, boxes);
        ref.current = element;
      }}>
        {boxes.map((_, index) => <span key={index} />)}
      </div>
      <output>{fitting}</output>
      <data>{slack}</data>
    </>
  );
};

describe("useFittingChildren", () => {
  it.each([
    { name: "every child on the first row", boxes: [[0, 0, 40], [0, 48, 40]] as [number, number, number][], fitting: "2", slack: "12" },
    { name: "a child wrapped to the second row", boxes: [[0, 0, 40], [16, 0, 40]] as [number, number, number][], fitting: "1", slack: "60" },
    { name: "a first child wider than the row", boxes: [[0, 0, 140]] as [number, number, number][], fitting: "0", slack: "100" },
  ])("counts $name and the width left after the last that fits", ({ boxes, fitting, slack }) => {
    const { container } = render(<Probe boxes={boxes} />);
    expect(container.querySelector("output")?.textContent).toBe(fitting);
    expect(container.querySelector("data")?.textContent).toBe(slack);
  });
});
