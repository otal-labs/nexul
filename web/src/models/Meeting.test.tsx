// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { Meeting } from "@nexul/client-core/calendar";

import {
  MeetingFormSchema,
  meetingBody,
  meetingFormDefaults,
  meetingPath,
  newMeetingDefaults,
  type MeetingFormData,
} from "@/models/Meeting";

const form = (over: Partial<MeetingFormData> = {}): MeetingFormData => ({ ...newMeetingDefaults(new Date(2026, 10, 2, 8, 20)), title: "Standup", ...over });

const issues = (data: MeetingFormData) => {
  const parsed = MeetingFormSchema.safeParse(data);
  return parsed.success ? [] : parsed.error.issues.map((i) => `${i.path.join(".")}: ${i.message}`);
};

describe("a new meeting's defaults", () => {
  it("start at the next full hour and run half an hour", () => {
    expect(newMeetingDefaults(new Date(2026, 10, 2, 8, 20))).toMatchObject({ date: "2026-11-02", start: "09:00", end: "09:30", remind_minutes: "10", where: "nowhere" });
  });

  it("roll into the next day late in the evening", () => {
    expect(newMeetingDefaults(new Date(2026, 10, 2, 23, 40))).toMatchObject({ date: "2026-11-03", start: "00:00", end: "00:30" });
  });
});

describe("the meeting form", () => {
  it("takes a title, a day and an end after the start", () => {
    expect(issues(form())).toEqual([]);
    expect(issues(form({ title: "  ", date: "", start: "", end: "" }))).toEqual(expect.arrayContaining(["title: Add a title", "date: Pick a day", "start: Pick a start", "end: Pick an end"]));
    expect(issues(form({ start: "10:00", end: "10:00" }))).toEqual(["end: End after the start"]);
  });

  it("wants a channel for a channel and an http link for a link, and nothing for nowhere", () => {
    expect(issues(form({ where: "channel" }))).toEqual(["voice_channel_id: Pick a channel"]);
    expect(issues(form({ where: "channel", voice_channel_id: "vc-1" }))).toEqual([]);
    expect(issues(form({ where: "link", link: "meet.example.com" }))).toEqual(["link: Use an http or https link"]);
    expect(issues(form({ where: "link", link: "https://meet.example.com/x" }))).toEqual([]);
    expect(issues(form({ where: "nowhere", voice_channel_id: "", link: "" }))).toEqual([]);
  });

  it("ends a repeat on or after the first day", () => {
    expect(issues(form({ repeats: "weekly", repeat_until: "2026-11-01" }))).toEqual(["repeat_until: Not before the first day"]);
    expect(issues(form({ repeats: "weekly", repeat_until: "2026-11-02" }))).toEqual([]);
  });
});

describe("meetingBody", () => {
  it("sends only the place that was picked", () => {
    const picked = { voice_channel_id: "vc-1", link: "https://meet.example.com/x" };
    expect(meetingBody(form({ ...picked, where: "channel" }), "change")).toMatchObject({ voice_channel_id: "vc-1", link: "" });
    expect(meetingBody(form({ ...picked, where: "link" }), "change")).toMatchObject({ voice_channel_id: "", link: "https://meet.example.com/x" });
    expect(meetingBody(form({ ...picked, where: "nowhere" }), "change")).toMatchObject({ voice_channel_id: "", link: "" });
  });

  it("names the browser's zone on a new meeting only, and leaves repeats off one occurrence", () => {
    const f = form({ repeats: "weekly", repeat_until: "2026-12-31", remind_minutes: "30", invitee_ids: ["u-2"] });
    const created = meetingBody(f, "create");
    expect(created).toMatchObject({ timezone: Intl.DateTimeFormat().resolvedOptions().timeZone, repeats: "weekly", remind_minutes: 30, invitee_ids: ["u-2"] });
    expect(created.repeat_until).toBeDefined();
    expect(meetingBody(f, "change")).not.toHaveProperty("timezone");
    expect(meetingBody(f, "one")).not.toHaveProperty("repeats");
    expect(meetingBody(f, "one")).not.toHaveProperty("repeat_until");
  });

  it("clears a series' end on a change and leaves a new meeting's out", () => {
    expect(meetingBody(form({ repeats: "weekly", repeat_until: "" }), "change")).toMatchObject({ repeat_until: "" });
    expect(meetingBody(form({ repeats: "weekly", repeat_until: "" }), "create")).not.toHaveProperty("repeat_until");
  });

  it("writes the day and times as the instants they name", () => {
    const body = meetingBody(form({ date: "2026-11-02", start: "09:00", end: "09:45" }), "create");
    expect(body.starts_at).toBe(new Date(2026, 10, 2, 9, 0).toISOString());
    expect(body.ends_at).toBe(new Date(2026, 10, 2, 9, 45).toISOString());
  });
});

const meeting: Meeting = {
  id: "m-1", workspace_id: "ws-1", organiser_id: "u-1", title: "Standup", notes: "Agenda", starts_at: "2026-11-02T10:00:00Z", ends_at: "2026-11-02T10:30:00Z",
  timezone: "UTC", voice_channel_id: "", link: "https://meet.example.com/x", repeats: "weekly", repeat_until: "2026-12-31T23:59:59Z", remind_minutes: 30,
  invitees: [{ user_id: "u-2", reply: "pending" }], created_at: "", updated_at: "",
};

describe("meetingFormDefaults", () => {
  it("fill the form from the meeting, the times from the occurrence being changed", () => {
    const defaults = meetingFormDefaults(meeting, "2026-11-09T10:00:00Z");
    const start = new Date("2026-11-09T10:00:00Z");
    expect(defaults).toMatchObject({ title: "Standup", where: "link", link: "https://meet.example.com/x", repeats: "weekly", remind_minutes: "30", invitee_ids: ["u-2"], notes: "Agenda" });
    expect(defaults.date).toBe(`${start.getFullYear()}-${String(start.getMonth() + 1).padStart(2, "0")}-${String(start.getDate()).padStart(2, "0")}`);
    expect(defaults.repeat_until).not.toBe("");
  });

  it("read a voice channel and nowhere", () => {
    expect(meetingFormDefaults({ ...meeting, link: "", voice_channel_id: "vc-1" }).where).toBe("channel");
    const endless: Meeting = { ...meeting, link: "" };
    delete endless.repeat_until;
    expect(meetingFormDefaults(endless)).toMatchObject({ where: "nowhere", repeat_until: "" });
  });
});

describe("meetingPath", () => {
  it("opens the calendar on the meeting", () => {
    expect(meetingPath("m 1")).toBe("/calendar?meeting=m%201");
  });
});
