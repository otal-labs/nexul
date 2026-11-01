import { describe, expect, it } from "vitest";

import { restingTilt, stepTilt, type Tilt } from "@/components/board/dragTilt";

// Moves the pointer at a steady speed (px per ms) for a while, one frame every frameMs.
const drag = (tilt: Tilt, speed: number, ms: number, frameMs = 16) => {
  let next = tilt;
  for (let t = 0; t < ms; t += frameMs) next = stepTilt(next, frameMs, next.x + speed * frameMs, null);
  return next;
};

describe("stepTilt", () => {
  it("leans the way the pointer moves and stays near the 5° cap on a wild fling", () => {
    let peak = 0;
    let tilt = restingTilt(0);
    for (let t = 0; t < 400; t += 16) {
      tilt = drag(tilt, 20, 16);
      peak = Math.max(peak, tilt.angle);
    }
    expect(drag(restingTilt(0), -1, 200).angle).toBeLessThan(0);
    expect(peak).toBeGreaterThan(4.5);
    expect(peak).toBeLessThan(6.5);
  });

  it("comes back level once the pointer holds still", () => {
    const still = drag(drag(restingTilt(0), 2, 300), 0, 1500);

    expect(Math.abs(still.angle)).toBeLessThan(0.01);
  });

  it("stays bounded when frames run long", () => {
    const slow = drag(restingTilt(0), 20, 1000, 50);

    expect(Math.abs(slow.angle)).toBeLessThan(6.5);
  });

  it("is level 200ms after release, inside the drop glide", () => {
    const held = drag(restingTilt(0), 3, 200);
    const landing = stepTilt(held, 16, held.x, 100);
    const landed = stepTilt(landing, 16, held.x, 200);

    expect(held.angle).toBeGreaterThan(1);
    expect(Math.abs(landing.angle)).toBeLessThan(Math.abs(held.angle));
    expect(landed.angle).toBe(0);
  });
});
