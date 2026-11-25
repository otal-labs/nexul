// @vitest-environment node
import { describe, expect, it } from "vitest";

import { pullRequestLink } from "@/utils/TrailTranscriptUtility";

describe("pullRequestLink", () => {
  it("finds the pull request the reply links", () => {
    expect(pullRequestLink("Opened https://github.com/otal-labs/nexul/pull/51 for review.")).toEqual({
      url: "https://github.com/otal-labs/nexul/pull/51",
      label: "otal-labs/nexul#51",
    });
    expect(pullRequestLink("No PR this time.")).toBeNull();
  });
});
