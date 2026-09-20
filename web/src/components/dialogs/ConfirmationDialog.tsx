import type { ConfirmDialogProps as ReactConfirmDialogProps } from "react-confirm";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export interface ConfirmationDialogOptions {
  message: string;
  title?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  destructive?: boolean;
}

export const ConfirmationDialog = ({
  show,
  proceed,
  message,
  title,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  destructive = true,
}: ReactConfirmDialogProps<ConfirmationDialogOptions, boolean>) => (
  <Dialog
    open={show}
    onOpenChange={(open) => {
      if (!open) proceed(false);
    }}
  >
    <DialogContent>
        <DialogHeader>
          <DialogTitle>{title ?? message}</DialogTitle>
          {title !== undefined && <DialogDescription>{message}</DialogDescription>}
        </DialogHeader>
      <DialogFooter>
        <Button variant="outline" onClick={() => proceed(false)}>
          {cancelLabel}
        </Button>
        <Button variant={destructive ? "destructive" : "default"} onClick={() => proceed(true)}>
          {confirmLabel}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
);

