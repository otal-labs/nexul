import { ContextAwareConfirmation, confirmable } from "react-confirm";

import { ConfirmationDialog, type ConfirmationDialogOptions } from "@/components/dialogs/ConfirmationDialog";

const confirm = ContextAwareConfirmation.createConfirmation<ConfirmationDialogOptions, boolean>(
  confirmable<ConfirmationDialogOptions, boolean>(ConfirmationDialog),
  // 0 = unmount as soon as it settles; the default delay leaves a stale entry a re-mount would resurrect.
  0,
);

export const useConfirmationDialog = () => ({
  open: confirm,
});
