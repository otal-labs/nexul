import { describe, expect, it } from "vitest";

import { loginHandle } from "@/models/User";

describe("loginHandle", () => {
  it("puts an @ before a username but not before an email login", () => {
    expect(loginHandle("lena")).toBe("@lena");
    expect(loginHandle("lena@example.com")).toBe("lena@example.com");
  });
});
