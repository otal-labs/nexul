import { describe, expect, test } from "bun:test";
import { loadConfig } from "../src/config.ts";

const base = {
  NEXUL_SERVER_URL: "https://nexul.example.com/",
  NEXUL_CREDENTIAL_FILE: "/opt/nexul/automations-jobs/credential",
  NEXUL_AUTOMATIONS_HOST_NAME: "jobs",
};

describe("loadConfig", () => {
  test.each(Object.keys(base))("requires %s", (name) => {
    expect(() => loadConfig({ ...base, [name]: "" })).toThrow(name);
  });

  test("reads the env contract with defaults", () => {
    const cfg = loadConfig({ ...base, NEXUL_AUTOMATIONS_POLL_MS: "nope" });
    expect(cfg.serverUrl).toBe("https://nexul.example.com");
    expect(cfg.ctl).toBe("nexul");
    expect(cfg.enrollCodeFile).toBeNull();
    expect(cfg.pollMs).toBe(10_000);
  });

  test("takes NEXUL_CTL and a dev stack's enroll code file", () => {
    const cfg = loadConfig({ ...base, NEXUL_CTL: "/usr/local/bin/nexul", NEXUL_ENROLL_CODE_FILE: "/data/enroll/automations-instance" });
    expect(cfg.ctl).toBe("/usr/local/bin/nexul");
    expect(cfg.enrollCodeFile).toBe("/data/enroll/automations-instance");
  });
});
