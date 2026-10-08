import {
  ArrowRightLeftIcon,
  AtSignIcon,
  BrainIcon,
  CircleHelpIcon,
  FilePenLineIcon,
  FilePlusIcon,
  MessageCircleQuestionIcon,
  MessageSquareReplyIcon,
  SparklesIcon,
  UserPlusIcon,
  type LucideIcon,
} from "lucide-react";

import { NotificationKind } from "@/models/Notification";

// What happened, at a glance down the list; the summary line still says it in words.
export const notificationKindIcon: Record<NotificationKind, LucideIcon> = {
  [NotificationKind.TicketAssigned]: UserPlusIcon,
  [NotificationKind.TicketMentioned]: AtSignIcon,
  [NotificationKind.TicketStatusChanged]: ArrowRightLeftIcon,
  [NotificationKind.DocCreated]: FilePlusIcon,
  [NotificationKind.DocUpdated]: FilePenLineIcon,
  [NotificationKind.DocMentioned]: AtSignIcon,
  [NotificationKind.DocQuestionsAsked]: MessageCircleQuestionIcon,
  [NotificationKind.DocQuestionsAnswered]: MessageSquareReplyIcon,
  [NotificationKind.MemoryUpdated]: BrainIcon,
  [NotificationKind.PlayRunFinished]: SparklesIcon,
  [NotificationKind.PlayRunWaiting]: CircleHelpIcon,
};
