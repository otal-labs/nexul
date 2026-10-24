import { FormInput } from "@/components/FormInput";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import type { MeetingFormData } from "@/models/Meeting";

// The day and the times it starts and ends, in the zone the browser runs in.
export const MeetingWhenFields = () => {
  const { control } = useFormDialogContext<MeetingFormData>();
  return (
    <div className="grid grid-cols-[1.4fr_1fr_1fr] items-start gap-3">
      <FormInput control={control} name="date" type="date" label="Day" />
      <FormInput control={control} name="start" type="time" label="Starts" />
      <FormInput control={control} name="end" type="time" label="Ends" />
    </div>
  );
};
