import { describe, expect, it } from "vitest";

import { getMeKey } from "@/hooks/AuthHooks";
import { getWelcomeKey, welcomeFollower } from "@/hooks/WelcomeHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("welcomeFollower", () => {
  const client = () => seeded([[[getMeKey], { user: { id: "u-me" } }], [[getWelcomeKey], { first_run_done: true, steps: [], dot: false }]]);

  it("refetches the viewer's progress when a frame names them", async () => {
    const c = client();
    await followFrame(welcomeFollower, "welcome.progress_changed", { user_id: "u-me" }, c);
    expect(isStale(c, [getWelcomeKey])).toBe(true);
  });

  it("leaves the viewer's progress alone for a frame about somebody on Team", async () => {
    const c = client();
    await followFrame(welcomeFollower, "welcome.progress_changed", { user_id: "u-other" }, c);
    expect(isStale(c, [getWelcomeKey])).toBe(false);
  });
});
