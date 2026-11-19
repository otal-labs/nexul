import { describe, expect, it } from "vitest";

import { isFollowableRedirect, redirectTarget } from "@/models/OAuth";

describe("redirectTarget", () => {
  it("names a site by its host and an app by its scheme too, so a scheme cannot pass for a known host", () => {
    expect(redirectTarget("https://agent.example.com/cb?x=1")).toBe("agent.example.com");
    expect(redirectTarget("http://127.0.0.1:4000/cb")).toBe("127.0.0.1:4000");
    expect(redirectTarget("web+evil://agent.example.com/cb")).toBe("web+evil://agent.example.com");
  });
});

describe("isFollowableRedirect", () => {
  it("follows a site or an app, never a script or a local document", () => {
    expect(isFollowableRedirect("https://agent.example.com/cb")).toBe(true);
    expect(isFollowableRedirect("com.example.agent://oauth/callback")).toBe(true);
    for (const to of ["javascript:alert(1)", "data:text/html,x", "not a url"]) expect(isFollowableRedirect(to)).toBe(false);
  });
});
