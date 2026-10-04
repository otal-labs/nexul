import type { ReactNode } from "react";

interface DialogPillProps {
  icon: ReactNode;
  label: string;
  trailing: ReactNode;
  onOpen: () => void;
  // A browser spaces the flex row's items apart when it names the button from them, so a pill that adds words names itself.
  name?: string;
}

// The attachment pill's shape, as a button that opens a dialog: an icon in a muted disc, the label, then trailing meta.
export const DialogPill = ({ icon, label, trailing, onOpen, name }: DialogPillProps) => (
  <button
    type="button"
    onClick={onOpen}
    aria-haspopup="dialog"
    aria-label={name}
    className="mx-1 inline-flex w-fit max-w-full items-center gap-1.5 rounded-full border border-border bg-card py-0.5 pr-2.5 pl-1 text-xs transition-colors duration-150 ease-standard hover:bg-accent/40"
  >
    <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted/60">{icon}</span>
    <span className="truncate text-foreground">{label}</span>
    {trailing}
  </button>
);
