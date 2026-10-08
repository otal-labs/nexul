import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { FormInput } from "@/components/FormInput";
import { InterviewSourcePicker } from "@/components/memory/InterviewSourcePicker";
import { InterviewSourceStance } from "@/components/memory/InterviewSourceStance";
import { InterviewSourceTextFields } from "@/components/memory/InterviewSourceTextFields";
import { useAddInterviewSource } from "@/hooks/InterviewSourceHooks";
import {
  AddSourceFormSchema,
  SOURCE_KIND_LABEL,
  SOURCE_KINDS,
  formStance,
  toAddSourceInput,
  type AddSourceFormData,
  type SourceKind,
  type SourceStance,
} from "@/models/InterviewSource";

interface InterviewSourceAddFormProps {
  projectId: string;
  onClose: () => void;
}

const STANCE_HINT: Record<SourceStance, string> = {
  follow: "The agent drafts answers from it.",
  question: "Never drafted from; the follow-ups ask about it.",
};

const kindClass = "h-7 px-2.5 text-xs text-muted-foreground data-[state=on]:bg-accent data-[state=on]:text-foreground";

// Adding a source inline: the kind, its field, and the stance preselected by kind until the person picks one.
export const InterviewSourceAddForm = ({ projectId, onClose }: InterviewSourceAddFormProps) => {
  const form = useForm<AddSourceFormData>({
    resolver: zodResolver(AddSourceFormSchema),
    defaultValues: { kind: "path", path: "", ref: "", label: "", body: "", stance: "" },
  });
  const add = useAddInterviewSource();
  const [kind, path, picked] = useWatch({ control: form.control, name: ["kind", "path", "stance"] });
  const stance = formStance({ kind, path, stance: picked });

  const pickKind = (next: string) => {
    if (next === "") return;
    form.setValue("kind", next as SourceKind);
    form.setValue("ref", "");
    form.clearErrors();
  };

  const submit = form.handleSubmit((values) => add.mutate(toAddSourceInput(projectId, values), { onSuccess: onClose }));

  return (
    <FormProvider {...form}>
      <form
        onSubmit={submit}
        aria-label="Add a source"
        className="animate-in fade-in-0 slide-in-from-top-1 space-y-4 rounded-lg border border-border p-4 duration-200 ease-out"
      >
        <ToggleGroup type="single" variant="outline" size="sm" aria-label="Kind of source" value={kind} onValueChange={pickKind} className="flex-wrap">
          {SOURCE_KINDS.map((k) => (
            <ToggleGroupItem key={k} value={k} className={kindClass}>
              {SOURCE_KIND_LABEL[k]}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        {kind === "path" && (
          <div className="space-y-2">
            <FormInput control={form.control} name="path" label="Path" placeholder="practices/ or docs/standards.md" className="font-mono text-[13px]" />
            <p className="text-xs text-muted-foreground">A file or folder, relative to the project's checkout.</p>
          </div>
        )}
        {kind !== "path" && kind !== "text" && <InterviewSourcePicker key={kind} kind={kind} projectId={projectId} />}
        {kind === "text" && <InterviewSourceTextFields />}
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <span className="text-sm font-medium">Stance</span>
          <InterviewSourceStance label="the new source" size="sm" value={stance} onChange={(v) => form.setValue("stance", v)} />
          <span className="min-w-0 flex-1 basis-48 text-xs text-muted-foreground">{STANCE_HINT[stance]}</span>
        </div>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" size="sm" loading={add.isPending}>
            Add
          </Button>
        </div>
      </form>
    </FormProvider>
  );
};
