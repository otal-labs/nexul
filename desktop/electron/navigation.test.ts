import { describe, expect, it } from "vitest";

import { isAllowedNavigation } from "./navigation";

const INSTANCE = "https://deploy.example.com";
const FILE_URL = "file:///home/dev/dist/index.html";

describe("isAllowedNavigation", () => {
  describe("while on the launcher (file:)", () => {
    it("allows staying on the launcher", () => {
      expect(isAllowedNavigation(FILE_URL, FILE_URL, null)).toBe(true);
    });

    it("allows navigating to the active instance origin", () => {
      expect(isAllowedNavigation(`${INSTANCE}/`, FILE_URL, INSTANCE)).toBe(true);
    });

    it("allows the GitHub OAuth dance", () => {
      expect(
        isAllowedNavigation("https://github.com/login/oauth/authorize?client_id=x", FILE_URL, INSTANCE),
      ).toBe(true);
    });

    it("blocks arbitrary external origins", () => {
      expect(isAllowedNavigation("https://evil.example/phish", FILE_URL, INSTANCE)).toBe(false);
    });

    it("blocks javascript: and about: URLs", () => {
      expect(isAllowedNavigation("javascript:alert(1)", FILE_URL, INSTANCE)).toBe(false);
      expect(isAllowedNavigation("about:blank", FILE_URL, INSTANCE)).toBe(false);
    });
  });

  describe("while connected to an instance", () => {
    const current = `${INSTANCE}/tickets`;

    it("allows same-origin SPA navigation", () => {
      expect(isAllowedNavigation(`${INSTANCE}/login`, current, INSTANCE)).toBe(true);
      expect(isAllowedNavigation(`${INSTANCE}/auth/github`, current, INSTANCE)).toBe(true);
    });

    it("allows the GitHub OAuth dance and its redirect back", () => {
      expect(isAllowedNavigation("https://github.com/login/oauth/authorize", current, INSTANCE)).toBe(true);
      expect(isAllowedNavigation(`${INSTANCE}/auth/callback?code=x`, current, INSTANCE)).toBe(true);
    });

    it("blocks file: URLs — nothing may escape the instance to local disk", () => {
      expect(isAllowedNavigation("file:///etc/passwd", current, INSTANCE)).toBe(false);
    });

    it("blocks a different http origin even when github paths are similar", () => {
      expect(isAllowedNavigation("https://githubusercontent.com/x", current, INSTANCE)).toBe(false);
    });

    it("blocks any other origin", () => {
      expect(isAllowedNavigation("https://attacker.example", current, INSTANCE)).toBe(false);
      expect(isAllowedNavigation("http://localhost:9000", current, INSTANCE)).toBe(false);
    });
  });

  describe("with no active instance", () => {
    it("still allows the launcher and github, blocks everything else", () => {
      expect(isAllowedNavigation(FILE_URL, FILE_URL, null)).toBe(true);
      expect(isAllowedNavigation("https://github.com/login", FILE_URL, null)).toBe(true);
      expect(isAllowedNavigation("https://random.example", FILE_URL, null)).toBe(false);
    });
  });

  it("rejects unparseable target URLs", () => {
    expect(isAllowedNavigation("", FILE_URL, null)).toBe(false);
    expect(isAllowedNavigation("not a url", FILE_URL, null)).toBe(false);
  });
});
