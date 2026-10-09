import {
  ArrowRightLeft,
  AtSign,
  Brain,
  CircleHelp,
  FilePenLine,
  FilePlus,
  MessageCircleQuestion,
  MessageSquareReply,
  Sparkles,
  UserPlus,
  type LucideIcon,
} from "lucide-react-native";

import { NotificationKind } from "@/models/Notification";

// The glyph for what happened, the same set as the web's Inbox.
export const notificationKindIcon: Record<NotificationKind, LucideIcon> = {
  [NotificationKind.TicketAssigned]: UserPlus,
  [NotificationKind.TicketMentioned]: AtSign,
  [NotificationKind.TicketStatusChanged]: ArrowRightLeft,
  [NotificationKind.DocCreated]: FilePlus,
  [NotificationKind.DocUpdated]: FilePenLine,
  [NotificationKind.DocMentioned]: AtSign,
  [NotificationKind.DocQuestionsAsked]: MessageCircleQuestion,
  [NotificationKind.DocQuestionsAnswered]: MessageSquareReply,
  [NotificationKind.MemoryUpdated]: Brain,
  [NotificationKind.PlayRunFinished]: Sparkles,
  [NotificationKind.PlayRunWaiting]: CircleHelp,
};
