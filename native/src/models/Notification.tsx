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
  ComputerMoveRequested: "computer.move_requested",
  ServiceStopped: "service.container_stopped",
  ServiceRecovered: "service.container_recovered",
  ErrorRateCrossed: "stack.error_rate_crossed",
  ErrorRateRecovered: "stack.error_rate_recovered",
} as const;

export type NotificationKind = (typeof NotificationKind)[keyof typeof NotificationKind];

export const SubjectType = {
  Ticket: "ticket",
  Doc: "doc",
  Memory: "memory",
  // The phone shows no computers; the notice marks read with nothing to open.
  Computer: "computer",
  Stack: "stack",
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
}

export interface UnreadCount {
  count: number;
}
