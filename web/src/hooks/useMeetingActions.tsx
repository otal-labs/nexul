import { toast } from "sonner";

import { asksScope, type Meeting } from "@nexul/client-core/calendar";

import { MeetingScopeForm } from "@/components/calendar/MeetingScopeForm";
import { errorMessage } from "@/api/client";
import { useUpdateMeeting, type MeetingChange } from "@/hooks/CalendarHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useMeetingFormDialog, useMoveToMeeting } from "@/hooks/useMeetingFormDialog";
import { MeetingScopeSchema, type MeetingScope, type MeetingScopeFormData } from "@/models/Meeting";

// Edit, Cancel and Restore on a meeting open in the panel, at the occurrence its slot names.
export const useMeetingActions = (meeting: Meeting, slot: string) => {
  const { open: openScope } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const { edit } = useMeetingFormDialog();
  const moveTo = useMoveToMeeting();
  const update = useUpdateMeeting();

  // A change to one occurrence of a repeating meeting asks which meetings it covers; any other covers the whole meeting.
  const chooseScope = async (okLabel: string, cancelLabel: string): Promise<MeetingScope | "all" | undefined> => {
    if (!slot || !asksScope(meeting)) return "all";
    const result = await openScope<MeetingScopeFormData>({
      title: "Which meetings?",
      schema: MeetingScopeSchema,
      form: <MeetingScopeForm />,
      formOptions: { defaultValues: { scope: "this" } },
      okLabel,
      cancelLabel,
    });
    return result.success ? result.data?.scope : undefined;
  };

  const change = async (body: MeetingChange, scope: MeetingScope | "all") => {
    try {
      const saved = await update.mutateAsync({ id: meeting.id, change: scope === "all" ? body : { ...body, slot, scope } });
      moveTo(saved, scope !== "all");
    } catch (error) {
      toast.error(errorMessage(error));
    }
  };

  return {
    pending: update.isPending,
    edit: async () => {
      const scope = await chooseScope("Continue", "Cancel");
      if (scope) await edit({ meeting, slot, scope });
    },
    cancel: async () => {
      const scope = await chooseScope("Cancel meeting", "Keep");
      if (!scope) return;
      const what = asksScope(meeting) ? "every meeting in this series" : "this meeting";
      const sure = scope !== "all" || (await confirm({ message: `Cancel ${what}? Everyone invited is told.`, confirmLabel: "Cancel meeting", cancelLabel: "Keep" }));
      if (sure) await change({ cancelled: true }, scope);
    },
    restore: () => change({ cancelled: false }, "all"),
  };
};
