import { describe, expect, it } from "vitest";

import { frameHandlers } from "@/hooks/liveFrameHandlers";
import { pushTopics } from "@/hooks/livePushTopics";
import pushed from "@/hooks/liveTopics.generated.json";

// Topics the server pushes that the browser leaves alone on purpose, each with why.
const ignoredTopics: Record<string, string> = {
  "ticket.assignee_changed": "a deprecated alias published beside ticket.developer_changed, which the browser follows",
};

const followed = new Set([...Object.keys(frameHandlers), ...Object.keys(pushTopics)]);

// The server's audience rules generate liveTopics.generated.json (make live-topics); a topic without a rule reaches
// nobody, so the two tables only work while they agree.
describe("the live topic contract", () => {
  it("follows no topic the server never pushes", () => {
    expect([...followed].filter((topic) => !pushed.includes(topic))).toEqual([]);
  });

  it("follows every pushed topic, or lists it as ignored", () => {
    expect(pushed.filter((topic) => !followed.has(topic) && !(topic in ignoredTopics))).toEqual([]);
  });

  it("ignores only topics that are pushed and followed nowhere", () => {
    expect(Object.keys(ignoredTopics).filter((topic) => followed.has(topic) || !pushed.includes(topic))).toEqual([]);
  });
});
