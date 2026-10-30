import type { ComponentType } from "react";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";

interface ListPaneEmptyProps {
  icon: ComponentType<{ className?: string }>;
  message: string;
  /** A line under the message saying what will land here. */
  hint?: string;
}

export const ListPaneEmpty = ({ icon, message, hint }: ListPaneEmptyProps) => (
  <EmptyState role="status" icon={icon} title={message} {...(hint && { message: hint })} size="compact" className="border-0" />
);

interface ListPaneNoMatchProps {
  onClear: () => void;
}

export const ListPaneNoMatch = ({ onClear }: ListPaneNoMatchProps) => (
  <EmptyState
    role="status"
    title="Nothing matches"
    size="compact"
    className="border-0"
    action={
      <Button variant="ghost" size="sm" onClick={onClear}>
        Clear search
      </Button>
    }
  />
);
