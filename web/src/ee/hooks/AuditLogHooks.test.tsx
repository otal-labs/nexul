// @vitest-environment node
import { describe, expect, it } from "vitest";

import { auditLogFollower, getAuditSettingsKey } from "@/ee/hooks/AuditLogHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("auditLogFollower", () => {
  it("refetches the settings when they change elsewhere", async () => {
    const client = seeded([[[getAuditSettingsKey], { retention_days: 45 }]]);

    await followFrame(auditLogFollower, "instance.audit_settings_changed", { retention_days: 365 }, client);

    expect(isStale(client, [getAuditSettingsKey])).toBe(true);
  });
});
