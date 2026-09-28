export const NotificationKind = {
  TicketAssigned: "ticket.assigned",
  TicketMentioned: "ticket.mentioned",
  TicketStatusChanged: "ticket.status_changed",
  DocCreated: "doc.created",
  DocUpdated: "doc.updated",
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
  kind: NotificationKind;
  subject_type: SubjectType;
  subject_id: string;
  subject_title: string;
  read: boolean;
  created_at: string;
}

export interface UnreadCount {
  count: number;
}
