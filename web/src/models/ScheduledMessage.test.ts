import { describe, expect, it } from "vitest";

import type { ScheduledMessage } from "@nexul/client-core/chat";

import { scheduleChange, scheduleFormDefaults } from "@/models/ScheduledMessage";

const weekly: ScheduledMessage = {
  id: "sm-1",
  conversation_id: "c-1",
  author_id: "u-1",
  body: "Standup notes",
  starts_at: "2026-11-02T09:00:00.000Z",
  send_at: "2026-11-23T09:00:00.000Z",
  repeats: "weekly",
  timezone: "UTC",
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};

describe("scheduleChange", () => {
  it("leaves the send time out when only the text changed, so a repeat keeps the day it steps from", () => {
    const change = scheduleChange(weekly, { ...scheduleFormDefaults(weekly), body: "Standup notes, moved" });
    expect(change).not.toHaveProperty("send_at");
    expect(change.body).toBe("Standup notes, moved");
  });

  it("sends the new time when it moved", () => {
    const form = { ...scheduleFormDefaults(weekly), date: "2026-11-24" };
    expect(scheduleChange(weekly, form).send_at).toBeDefined();
  });
});
