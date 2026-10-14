import { CircleHelp } from "lucide-react";

import { RemoveLinkButton } from "@/components/ticket/RemoveLinkButton";

interface OriginUnknownRowProps {
  projectId: string;
  onRemove: () => void;
}

// The found-in marker a reporter sets when nobody knows which ticket a bug came from.
export const OriginUnknownRow = ({ projectId, onRemove }: OriginUnknownRowProps) => (
  <li className="group flex items-center gap-2 px-2 py-1 text-sm text-muted-foreground">
    <CircleHelp className="size-3.5 shrink-0" aria-hidden />
    <span className="flex-1 py-1">Origin unknown</span>
    <RemoveLinkButton projectId={projectId} label="Remove origin unknown" onRemove={onRemove} />
  </li>
);
