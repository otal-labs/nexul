import { NotificationKind, SubjectType, type Notification } from "@/models/Notification";

const kindLabels: Record<NotificationKind, string> = {
  [NotificationKind.TicketAssigned]: "assigned to you",
  [NotificationKind.TicketMentioned]: "mention",
  [NotificationKind.TicketStatusChanged]: "status changed",
  [NotificationKind.DocCreated]: "doc created",
  [NotificationKind.DocUpdated]: "doc updated",
  [NotificationKind.DocMentioned]: "mention",
  [NotificationKind.DocQuestionsAsked]: "new questions",
  [NotificationKind.DocQuestionsAnswered]: "questions answered",
  [NotificationKind.MemoryUpdated]: "memory updated",
  [NotificationKind.PlayRunFinished]: "play run ended",
  [NotificationKind.PlayRunWaiting]: "needs your answer",
};

/** One inbox row: a doc with every notification about it, or any other notification on its own. */
export interface InboxRow {
  type: "doc" | "notification";
  key: string;
  /** The newest notification, which places the row and is what the detail pane opens. */
  notification: Notification;
  title: string;
  summary: string;
  unreadIds: string[];
}

/** The doc rows of one non-default folder, placed by the newest of them. */
export interface InboxFolder {
  type: "folder";
  key: string;
  name: string;
  rows: InboxRow[];
  notificationCount: number;
  /** "4 docs · 7 updates", every notification in the group counted as an update. */
  summary: string;
  unread: boolean;
}

export type InboxEntry = InboxRow | InboxFolder;

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? "" : "s"}`;

const docSummary = (ns: Notification[]) => {
  const count = (kind: NotificationKind) => ns.filter((n) => n.kind === kind).length;
  const updates = count(NotificationKind.DocUpdated);
  const mentions = count(NotificationKind.DocMentioned);
  const parts = [
    count(NotificationKind.DocCreated) > 0 && "created",
    updates === 1 && "updated",
    updates > 1 && plural(updates, "update"),
    mentions === 1 && "mentioned you",
    mentions > 1 && `mentioned you ${mentions} times`,
    count(NotificationKind.DocQuestionsAsked) > 0 && kindLabels[NotificationKind.DocQuestionsAsked],
    count(NotificationKind.DocQuestionsAnswered) > 0 && kindLabels[NotificationKind.DocQuestionsAnswered],
  ];
  return parts.filter(Boolean).join(" · ");
};

// "GetSource EP07: Bluesky" under GetSource reads "EP07: Bluesky"; a title that is only the name stays whole.
const stripFolderPrefix = (title: string, folder: string) => {
  const escaped = folder.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const rest = title.replace(new RegExp(`^${escaped}[\\s:\\-–—]+`, "i"), "");
  return rest === "" ? title : rest;
};

const rowOf = (ns: Notification[], folderName: string | undefined): InboxRow => {
  const newest = ns[0] as Notification;
  const unreadIds = ns.filter((n) => !n.read).map((n) => n.id);
  if (newest.subject_type !== SubjectType.Doc) {
    return { type: "notification", key: `n:${newest.id}`, notification: newest, title: newest.subject_title, summary: kindLabels[newest.kind], unreadIds };
  }
  // These kinds word their title around the doc's, so the doc's own title comes from any other notification when there is one.
  const titleWorded: NotificationKind[] = [NotificationKind.DocMentioned, NotificationKind.DocQuestionsAsked, NotificationKind.DocQuestionsAnswered];
  const title = (ns.find((n) => !titleWorded.includes(n.kind)) ?? newest).subject_title;
  return {
    type: "doc",
    key: `doc:${newest.subject_id}`,
    notification: newest,
    title: folderName ? stripFolderPrefix(title, folderName) : title,
    summary: docSummary(ns),
    unreadIds,
  };
};

const groupedFolderId = (n: Notification) =>
  n.subject_type === SubjectType.Doc && n.folder_id && !n.folder_is_default ? n.folder_id : undefined;

/**
 * Groups an inbox, newest first as the server lists it, into rows and folder groups: one row per doc, docs of a
 * non-default folder under that folder, and every other notification on its own row, all ordered by newest activity.
 */
export const groupInbox = (notifications: Notification[]): InboxEntry[] => {
  const newestFirst = [...notifications].sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at));
  const docs = new Map<string, Notification[]>();
  for (const n of newestFirst) {
    if (n.subject_type !== SubjectType.Doc) continue;
    docs.set(n.subject_id, [...(docs.get(n.subject_id) ?? []), n]);
  }

  const entries: InboxEntry[] = [];
  const folders = new Map<string, InboxFolder>();
  for (const n of newestFirst) {
    if (n.subject_type !== SubjectType.Doc) {
      entries.push(rowOf([n], undefined));
      continue;
    }
    const ns = docs.get(n.subject_id);
    if (!ns || ns[0] !== n) continue;
    const folderId = groupedFolderId(n);
    if (!folderId) {
      entries.push(rowOf(ns, undefined));
      continue;
    }
    const name = n.folder_name ?? "";
    const row = rowOf(ns, name);
    const folder = folders.get(folderId);
    if (folder) {
      folder.rows.push(row);
      folder.notificationCount += ns.length;
      folder.unread ||= row.unreadIds.length > 0;
      continue;
    }
    const created: InboxFolder = { type: "folder", key: `folder:${folderId}`, name, rows: [row], notificationCount: ns.length, summary: "", unread: row.unreadIds.length > 0 };
    folders.set(folderId, created);
    entries.push(created);
  }
  for (const folder of folders.values()) {
    folder.summary = `${plural(folder.rows.length, "doc")} · ${plural(folder.notificationCount, "update")}`;
  }
  return entries;
};

/** Every row in display order, a folder group's rows in its place. */
export const inboxRows = (entries: InboxEntry[]): InboxRow[] =>
  entries.flatMap((e) => (e.type === "folder" ? e.rows : [e]));
