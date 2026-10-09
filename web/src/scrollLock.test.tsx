import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import css from "@/index.css?raw";

// jsdom applies no stylesheet, so the guard pairs what the real scroll lock writes with what index.css registers.
describe("modal scroll lock", () => {
  it("sets only custom properties that index.css registers as not inherited", async () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogTitle>New ticket</DialogTitle>
          <DialogDescription>Body</DialogDescription>
        </DialogContent>
      </Dialog>,
    );
    await screen.findByRole("dialog");

    const lockStyles = [...document.head.querySelectorAll("style")].map((s) => s.textContent ?? "").join("\n");
    const lockedBodyRules = [...lockStyles.matchAll(/body\[data-scroll-locked\]\s*\{([^}]*)\}/g)].map((m) => m[1]).join("\n");
    const variables = [...lockedBodyRules.matchAll(/(--[\w-]+)\s*:/g)].map((m) => m[1]!);

    // An inherited custom property on body restyles every element on the page each time a dialog or menu opens.
    expect(variables).not.toHaveLength(0);
    for (const variable of variables) {
      expect(css).toMatch(new RegExp(`@property ${variable} \\{[^}]*inherits: false;`));
    }
  });
});
