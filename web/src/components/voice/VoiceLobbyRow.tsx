import type { ReactNode } from "react";

interface VoiceLobbyRowProps {
  lead: ReactNode;
  title: ReactNode;
  detail?: ReactNode;
  actions: ReactNode;
}

// Every state before and around joining sits in this one slot, so switching states never moves the button.
export const VoiceLobbyRow = ({ lead, title, detail, actions }: VoiceLobbyRowProps) => (
  <div className="mx-auto w-full max-w-3xl shrink-0 px-3 pt-3">
    <div role="status" className="flex min-h-16 items-center gap-3 rounded-lg bg-surface-2 px-3 py-2.5">
      {lead}
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{title}</p>
        {detail && <div className="truncate text-xs text-muted-foreground">{detail}</div>}
      </div>
      <div className="flex shrink-0 items-center gap-1.5">{actions}</div>
    </div>
  </div>
);

// The row's empty slot while the voice code loads, so the chat below doesn't jump when the row arrives.
export const VoiceLobbySlot = () => (
  <div aria-hidden className="mx-auto w-full max-w-3xl shrink-0 px-3 pt-3">
    <div className="min-h-16 rounded-lg bg-surface-2" />
  </div>
);
