import { XIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";

interface RemoveLinkButtonProps {
  // The project of the ticket whose link goes.
  projectId: string | undefined;
  label: string;
  onRemove: () => void;
}

// The × at the end of a link row in the ticket's rail, shown on the row's hover and focus.
export const RemoveLinkButton = ({ projectId, label, onRemove }: RemoveLinkButtonProps) => (
  <ActionButton
    permission="tickets:write"
    projectId={projectId}
    verb="change links"
    variant="ghost"
    size="icon"
    icon={XIcon}
    aria-label={label}
    onClick={onRemove}
    className="size-7 shrink-0 text-muted-foreground opacity-0 transition-[color,background-color,opacity] duration-150 ease-standard group-focus-within:opacity-100 group-hover:opacity-100 [&_svg]:size-3.5"
  />
);
