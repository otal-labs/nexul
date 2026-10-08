import { useState, type DragEvent } from "react";
import { Upload } from "lucide-react";
import { useFormContext } from "react-hook-form";

import { Textarea } from "@/components/ui/textarea";
import { FormInput } from "@/components/FormInput";
import { isReadableTextFile, type AddSourceFormData } from "@/models/InterviewSource";

// Pasted text: a label and the text; a dropped or chosen text or markdown file is read here as if pasted, anything else refused.
export const InterviewSourceTextFields = () => {
  const { control, register, setValue, getValues, formState } = useFormContext<AddSourceFormData>();
  const [refused, setRefused] = useState("");
  const bodyError = formState.errors.body?.message;

  const take = async (file: File | undefined) => {
    if (!file) return;
    if (!isReadableTextFile(file)) {
      setRefused(`${file.name} isn't text or markdown. Paste its text instead.`);
      return;
    }
    setRefused("");
    setValue("body", await file.text(), { shouldValidate: true });
    if (getValues("label").trim() === "") setValue("label", file.name, { shouldValidate: true });
  };

  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    void take(e.dataTransfer.files[0]);
  };

  return (
    <div className="space-y-3" onDragOver={(e) => e.preventDefault()} onDrop={onDrop}>
      <FormInput control={control} name="label" label="Label" placeholder="Handover notes" />
      <div className="space-y-2">
        <label htmlFor="source-body" className="text-sm font-medium">
          Text
        </label>
        <Textarea
          id="source-body"
          rows={5}
          aria-invalid={bodyError !== undefined}
          className="field-sizing-content max-h-64 min-h-24 resize-none"
          {...register("body")}
        />
        {bodyError && (
          <p role="alert" className="text-sm text-destructive">
            {bodyError}
          </p>
        )}
        <label className="flex w-fit cursor-pointer items-center gap-2 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground">
          <Upload className="size-3.5" aria-hidden />
          or drop a text or markdown file
          <input type="file" accept=".md,.markdown,.txt,text/*" className="sr-only" onChange={(e) => void take(e.target.files?.[0])} />
        </label>
        {refused !== "" && (
          <p role="alert" className="text-xs text-destructive">
            {refused}
          </p>
        )}
      </div>
    </div>
  );
};
