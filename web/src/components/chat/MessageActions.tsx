import { Pencil, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";

interface MessageActionsProps {
  onEdit: () => void;
  onDelete: () => void;
}

// A pill pinned to the top-right corner of its bubble; the row's hover or focus reveals it.
export const MessageActions = ({ onEdit, onDelete }: MessageActionsProps) => (
  <div className="pointer-events-none absolute -top-3.5 -right-2 z-10 flex rounded-md border border-input bg-popover opacity-0 shadow-card transition-opacity duration-150 ease-standard group-focus-within:pointer-events-auto group-focus-within:opacity-100 group-hover:pointer-events-auto group-hover:opacity-100">
    <Button size="icon" variant="ghost" className="size-6" aria-label="Edit message" onClick={onEdit}>
      <Pencil className="size-3.5" aria-hidden />
    </Button>
    <Button size="icon" variant="ghost" className="size-6" aria-label="Delete message" onClick={onDelete}>
      <Trash2 className="size-3.5" aria-hidden />
    </Button>
  </div>
);
