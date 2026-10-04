export const NotificationKind = {
  TicketAssigned: "ticket.assigned",
  TicketMentioned: "ticket.mentioned",
  TicketStatusChanged: "ticket.status_changed",
  DocCreated: "doc.created",
  DocUpdated: "doc.updated",
  DocMentioned: "doc.mentioned",
  DocQuestionsAsked: "doc.questions_asked",
  DocQuestionsAnswered: "doc.questions_answered",
  MemoryUpdated: "memory.updated",
  PlayRunFinished: "play.run_finished",
  PlayRunWaiting: "play.run_waiting",
} as const;

export type NotificationKind = (typeof NotificationKind)[keyof typeof NotificationKind];

export const SubjectType = {
  Ticket: "ticket",
  Doc: "doc",
  Memory: "memory",
} as const;

export type SubjectType = (typeof SubjectType)[keyof typeof SubjectType];

export interface Notification {
  id: string;
  user_id: string;
  workspace_id: string;
  kind: NotificationKind;
  subject_type: SubjectType;
  subject_id: string;
  subject_title: string;
  read: boolean;
  created_at: string;
  /** A doc's folder as it is now; absent for tickets and memories. */
  folder_id?: string;
  folder_name?: string;
  folder_is_default?: boolean;
}

export interface UnreadCount {
  count: number;
  /** Unread per workspace id; a workspace with none is absent. */
  workspaces: Record<string, number>;
}
