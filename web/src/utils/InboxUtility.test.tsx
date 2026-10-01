import { describe, expect, it } from "vitest";

import type { Notification } from "@/models/Notification";
import { groupInbox, inboxRows, type InboxFolder, type InboxRow } from "@/utils/InboxUtility";

let seq = 0;
const note = (over: Partial<Notification>): Notification => ({
  id: `n${++seq}`,
  user_id: "u1",
  workspace_id: "ws-1",
  kind: "doc.updated",
  subject_type: "doc",
  subject_id: "doc-1",
  subject_title: "Spec",
  read: true,
  created_at: "2026-10-01T10:00:00Z",
  ...over,
});

const inGetSource = { folder_id: "f-gs", folder_name: "GetSource", folder_is_default: false } as const;
const inMain = { folder_id: "f-main", folder_name: "Main", folder_is_default: true } as const;

const at = (hour: number) => `2026-10-01T${String(hour).padStart(2, "0")}:00:00Z`;

describe("groupInbox", () => {
  it("collapses every notification about one doc into a single row placed by its newest", () => {
    const entries = groupInbox([
      note({ id: "u2", kind: "doc.updated", subject_id: "d-spec", created_at: at(12), ...inMain }),
      note({ id: "t1", kind: "ticket.assigned", subject_type: "ticket", subject_id: "t-1", subject_title: "Fix it", created_at: at(11) }),
      note({ id: "u1", kind: "doc.updated", subject_id: "d-spec", created_at: at(10), ...inMain }),
      note({ id: "c1", kind: "doc.created", subject_id: "d-spec", created_at: at(9), ...inMain }),
    ]);

    expect(entries.map((e) => e.key)).toEqual(["doc:d-spec", "n:t1"]);
    const doc = entries[0] as InboxRow;
    expect(doc.type).toBe("doc");
    expect(doc.notification.id).toBe("u2");
    expect(doc.title).toBe("Spec");
    expect(doc.summary).toBe("created · 2 updates");
  });

  it("says who mentioned you beside the doc's own changes", () => {
    const [doc] = groupInbox([
      note({ kind: "doc.mentioned", subject_title: "Ada mentioned you in Spec", created_at: at(12), ...inMain }),
      note({ kind: "doc.updated", created_at: at(11), ...inMain }),
    ]) as InboxRow[];

    expect(doc?.summary).toBe("updated · mentioned you");
    expect(doc?.title).toBe("Spec");
  });

  it("puts docs of a non-default folder under one folder group, docs in Main stay plain rows", () => {
    const entries = groupInbox([
      note({ subject_id: "d-ep07", subject_title: "GetSource EP07: Bluesky feeds", created_at: at(12), ...inGetSource }),
      note({ subject_id: "d-main", subject_title: "Roadmap", created_at: at(11), ...inMain }),
      note({ subject_id: "d-ep06", subject_title: "GetSource EP06: Threads", kind: "doc.created", created_at: at(10), ...inGetSource }),
      note({ subject_id: "d-ep06", subject_title: "GetSource EP06: Threads", created_at: at(9), ...inGetSource }),
    ]);

    expect(entries.map((e) => e.key)).toEqual(["folder:f-gs", "doc:d-main"]);
    const folder = entries[0] as InboxFolder;
    expect(folder.name).toBe("GetSource");
    expect(folder.rows.map((r) => r.key)).toEqual(["doc:d-ep07", "doc:d-ep06"]);
    expect(folder.summary).toBe("2 docs · 3 updates");
    expect((entries[1] as InboxRow).title).toBe("Roadmap");
  });

  it("drops the folder name from the start of a title inside its group, ignoring case", () => {
    const titles = (groupInbox([
      note({ subject_id: "a", subject_title: "GetSource EP07: Bluesky feeds", ...inGetSource }),
      note({ subject_id: "b", subject_title: "getsource: EP08", ...inGetSource }),
      note({ subject_id: "c", subject_title: "GETSOURCE - EP09", ...inGetSource }),
      note({ subject_id: "d", subject_title: "GetSourcery notes", ...inGetSource }),
      note({ subject_id: "e", subject_title: "GetSource", ...inGetSource }),
    ])[0] as InboxFolder).rows.map((r) => r.title);

    expect(titles).toEqual(["EP07: Bluesky feeds", "EP08", "EP09", "GetSourcery notes", "GetSource"]);
  });

  it("keeps a title whole outside a group", () => {
    const [row] = groupInbox([note({ subject_title: "Main notes", folder_id: "f-main", folder_name: "Main", folder_is_default: true })]) as InboxRow[];
    expect(row?.title).toBe("Main notes");
  });

  it("rolls unread up from notification to doc row to folder group", () => {
    const entries = groupInbox([
      note({ id: "r1", subject_id: "d-a", read: true, created_at: at(12), ...inGetSource }),
      note({ id: "x1", subject_id: "d-b", read: false, created_at: at(11), ...inGetSource }),
      note({ id: "x2", subject_id: "d-b", kind: "doc.created", read: false, created_at: at(10), ...inGetSource }),
      note({ id: "r2", subject_id: "d-c", read: true, created_at: at(9), folder_id: "f-old", folder_name: "Old", folder_is_default: false }),
    ]);

    const [gs, old] = entries as InboxFolder[];
    expect(gs?.unread).toBe(true);
    expect(gs?.rows.map((r) => r.unreadIds)).toEqual([[], ["x1", "x2"]]);
    expect(old?.unread).toBe(false);
  });

  it("orders groups, doc rows and other notifications together by newest activity", () => {
    const entries = groupInbox([
      note({ id: "m1", subject_type: "memory", kind: "memory.updated", subject_id: "mem-1", created_at: at(13) }),
      note({ subject_id: "d-ep01", created_at: at(12), ...inGetSource }),
      note({ subject_id: "d-main", created_at: at(11), ...inMain }),
      note({ subject_id: "d-ep02", created_at: at(10), ...inGetSource }),
      note({ id: "t1", subject_type: "ticket", kind: "ticket.status_changed", subject_id: "t-1", created_at: at(9) }),
    ]);

    expect(entries.map((e) => e.key)).toEqual(["n:m1", "folder:f-gs", "doc:d-main", "n:t1"]);
  });

  it("leaves every non-doc notification as its own row, even about the same ticket", () => {
    const entries = groupInbox([
      note({ id: "t1", subject_type: "ticket", kind: "ticket.status_changed", subject_id: "t-1", subject_title: "Fix it", read: false }),
      note({ id: "t2", subject_type: "ticket", kind: "ticket.assigned", subject_id: "t-1", subject_title: "Fix it" }),
    ]) as InboxRow[];

    expect(entries.map((e) => [e.type, e.notification.id, e.title, e.summary, e.unreadIds])).toEqual([
      ["notification", "t1", "Fix it", "status changed", ["t1"]],
      ["notification", "t2", "Fix it", "assigned to you", []],
    ]);
  });

  it("names what happened to a memory or a play run", () => {
    const entries = groupInbox([
      note({ subject_type: "memory", kind: "memory.updated", subject_id: "mem-1", created_at: at(12) }),
      note({ subject_type: "ticket", kind: "play.run_finished", subject_id: "t-1", created_at: at(11) }),
      note({ subject_type: "ticket", kind: "play.run_waiting", subject_id: "t-1", created_at: at(10) }),
    ]) as InboxRow[];
    expect(entries.map((e) => e.summary)).toEqual(["memory updated", "play run ended", "needs your answer"]);
  });

  it("treats a doc whose folder is unknown, deleted docs included, as a plain row", () => {
    const entries = groupInbox([note({ subject_id: "gone" })]);
    expect(entries.map((e) => e.key)).toEqual(["doc:gone"]);
  });
});

describe("inboxRows", () => {
  it("lists every row in display order, a group's rows in place of the group", () => {
    const entries = groupInbox([
      note({ subject_id: "d-ep01", created_at: at(12), ...inGetSource }),
      note({ id: "t1", subject_type: "ticket", subject_id: "t-1", created_at: at(11) }),
    ]);
    expect(inboxRows(entries).map((r) => r.key)).toEqual(["doc:d-ep01", "n:t1"]);
  });
});
