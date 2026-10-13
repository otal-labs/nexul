import { describe, expect, it } from "vitest";

import pushed from "@/hooks/liveTopics.generated.json";
import { followedTopics as followed } from "@/hooks/useLiveEvents";

// Topics the server pushes that the browser leaves alone on purpose, each with why.
const ignoredTopics: Record<string, string> = {
  "ticket.assignee_changed": "a deprecated alias published beside ticket.developer_changed, which the browser follows",
};

// The server's audience rules generate liveTopics.generated.json (make live-topics); a topic without a rule reaches
// nobody, so the followers only work while they and the server agree.
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
