import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import { chatDayLabel } from "@/utils/ChatDayUtility";

export const ChatDayDivider = ({ createdAt }: { createdAt: string }) => (
  <div
    role="separator"
    aria-label={chatDayLabel(createdAt)}
    className={cn(microheaderClass, "flex items-center gap-3 px-3 pt-5 pb-1 before:h-px before:flex-1 before:bg-border after:h-px after:flex-1 after:bg-border")}
  >
    {chatDayLabel(createdAt)}
  </div>
);
