import { describe, expect, it } from "vitest";

import { unknownPerson } from "@nexul/client-core/person";

import type { Message } from "@/models/Chat";
import { runLabels } from "@/utils/ChatRunUtility";

const message = (id: string, minute: number, overrides: Partial<Message> = {}): Message => ({
  id,
  conversation_id: "c1",
  author_id: "bob",
  author_kind: "user",
  body: id,
  mentions: null,
  created_at: `2026-10-08T16:${String(minute).padStart(2, "0")}:00Z`,
  updated_at: `2026-10-08T16:${String(minute).padStart(2, "0")}:00Z`,
  ...overrides,
});

describe("runLabels", () => {
  it("names every message of a run by the message that opened it, and none for your own or a system line", () => {
    const bot = { author_kind: "bot" as const, author_id: "b1", author_name: "Deploy Notifier" };
    const labels = runLabels(
      [
        message("deploy", 0, bot),
        message("failed", 1, bot),
        message("mine", 2, { author_id: "me" }),
        message("bob", 3),
        message("joined", 4, { author_kind: "system", author_id: "" }),
      ],
      unknownPerson,
      "me",
    );
    const day = labels[0]?.day;
    expect(labels).toEqual([{ name: "Deploy Notifier", day }, { name: "Deploy Notifier", day }, undefined, { name: "bob", day }, undefined]);
  });
});
