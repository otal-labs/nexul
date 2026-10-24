import { REPEAT_LABELS } from "@nexul/client-core/calendar";

import { FormInput } from "@/components/FormInput";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormSelect } from "@/components/ticket/FormSelect";
import type { MeetingFormData } from "@/models/Meeting";

const REPEAT_OPTIONS = Object.entries(REPEAT_LABELS).map(([value, label]) => ({ value, label }));

// How often it comes round and the last day it does.
export const MeetingRepeatFields = () => {
  const { control, watch } = useFormDialogContext<MeetingFormData>();
  const repeats = watch("repeats");
  return (
    <div className="grid grid-cols-2 items-start gap-3">
      <FormSelect control={control} name="repeats" label="Repeats" options={REPEAT_OPTIONS} />
      {repeats !== "none" && <FormInput control={control} name="repeat_until" type="date" label="Until" />}
    </div>
  );
};
