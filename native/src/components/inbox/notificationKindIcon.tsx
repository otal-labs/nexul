import ArrowRightLeft from "lucide-react-native/icons/arrow-right-left";
import AtSign from "lucide-react-native/icons/at-sign";
import Brain from "lucide-react-native/icons/brain";
import CircleHelp from "lucide-react-native/icons/circle-question-mark";
import FilePenLine from "lucide-react-native/icons/file-pen-line";
import FilePlus from "lucide-react-native/icons/file-plus";
import MessageCircleQuestion from "lucide-react-native/icons/message-circle-question-mark";
import MessageSquareReply from "lucide-react-native/icons/message-square-reply";
import Sparkles from "lucide-react-native/icons/sparkles";
import UserPlus from "lucide-react-native/icons/user-plus";
import type { LucideIcon } from "lucide-react-native";

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
