import type { ReactNode } from "react";

import type { NoteVariant } from "@/components/note/noteVariants";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Sheet, SheetContent, SheetDescription, SheetTitle } from "@/components/ui/sheet";

interface NoteDialogProps {
  kind: NoteVariant["dialog"];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  children: ReactNode;
}

// Prototype only: the three frames a note's file opens in.
export const NoteDialog = ({ kind, open, onOpenChange, title, children }: NoteDialogProps) => (
  <>
    {kind !== "sheet" && (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className={
            kind === "tall"
              ? "flex h-[calc(100dvh-3rem)] flex-col bg-popover sm:max-w-5xl"
              : "max-h-[calc(100dvh-4rem)] overflow-y-auto bg-popover sm:max-w-3xl"
          }
        >
          <DialogTitle className="sr-only">{title}</DialogTitle>
          <DialogDescription className="sr-only">The note's file</DialogDescription>
          {children}
        </DialogContent>
      </Dialog>
    )}
    {kind === "sheet" && (
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent className="w-full overflow-y-auto bg-popover p-6 sm:max-w-2xl">
          <SheetTitle className="sr-only">{title}</SheetTitle>
          <SheetDescription className="sr-only">The note's file</SheetDescription>
          {children}
        </SheetContent>
      </Sheet>
    )}
  </>
);
