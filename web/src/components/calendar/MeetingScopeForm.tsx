import { Controller } from "react-hook-form";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import type { MeetingScopeFormData } from "@/models/Meeting";

const CHOICES = [
  { value: "this", label: "This meeting" },
  { value: "following", label: "This and following meetings" },
] as const;

// Which meetings of a series a change covers.
export const MeetingScopeForm = () => {
  const { control } = useFormDialogContext<MeetingScopeFormData>();
  return (
    <Controller
      control={control}
      name="scope"
      render={({ field }) => (
        <RadioGroup value={field.value} onValueChange={field.onChange} aria-label="Which meetings">
          {CHOICES.map((choice) => (
            <label key={choice.value} className="flex cursor-pointer items-center gap-3 text-sm">
              <RadioGroupItem value={choice.value} />
              {choice.label}
            </label>
          ))}
        </RadioGroup>
      )}
    />
  );
};
