import { readFileSync } from "node:fs";
import { join } from "node:path";

import { cn } from "@/lib/utils";

// Every size step global.css defines, read from the file so a new step can't be missed here.
const steps = [...readFileSync(join(__dirname, "../global.css"), "utf8").matchAll(/--text-([a-z]+(?:-(?!-)[a-z]+)*):/g)].map((m) => m[1]!);

test("the type scale is read from global.css", () => {
  expect(steps).toEqual(expect.arrayContaining(["micro", "row", "body", "screen-title"]));
});

// A step merged away as a colour took Text's text-foreground with it, so dark mode drew titles black on black.
test.each(steps)("text-%s keeps the colour beside it and replaces another size", (step) => {
  expect(cn("text-foreground text-base", `text-${step}`)).toBe(`text-foreground text-${step}`);
});
