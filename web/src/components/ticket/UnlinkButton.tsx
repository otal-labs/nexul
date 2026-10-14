import { XIcon } from "lucide-react";

import { useHasPermission } from "@/hooks/WorkspaceHooks";

interface UnlinkButtonProps {
  /** What it unlinks, for its name: "branch ticket/1". */
  subject: string;
  onUnlink: () => void;
}

// Shown on hover and focus, and only to someone who may edit the ticket's links.
export const UnlinkButton = ({ subject, onUnlink }: UnlinkButtonProps) => {
  const canWrite = useHasPermission("tickets:write");
  if (!canWrite) return null;
  return (
    <button
      type="button"
      aria-label={`Unlink ${subject}`}
      title={`Unlink ${subject}`}
      onClick={onUnlink}
      className="flex size-5 shrink-0 items-center justify-center rounded-md text-muted-foreground opacity-0 transition-[color,background-color,opacity] duration-150 ease-standard group-focus-within:opacity-100 group-hover:opacity-100 hover:bg-muted/50 hover:text-foreground"
    >
      <XIcon className="size-3" aria-hidden />
    </button>
  );
};
