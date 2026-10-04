import { useForm } from "react-hook-form";

import { FormCombobox } from "@/components/FormCombobox";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { SourcesProtoTextFields } from "@/components/memory/prototype/SourcesProtoTextFields";
import { KIND_LABEL, SourceStance } from "@/components/memory/prototype/SourcesProtoStance";
import { PASTED, PICKABLE, type SourceKind, type Stance } from "@/components/memory/prototype/SourcesProtoData";
import { useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";

export interface AddSourceValues {
  kind: SourceKind;
  path: string;
  ref: string;
  label: string;
  body: string;
  stance: Stance;
}

const KINDS: SourceKind[] = ["path", "doc", "memory", "project", "text"];

const defaultStance = (kind: SourceKind, path: string): Stance => {
  if (kind === "project") return "question";
  if (kind === "path") return path.trim().toLowerCase().endsWith(".md") ? "follow" : "question";
  return "follow";
};

const STANCE_HINT: Record<Stance, string> = {
  follow: "The agent drafts answers from it.",
  question: "Never drafted from; the follow-ups ask about it.",
};

const itemClass = "h-7 px-2.5 text-xs text-muted-foreground data-[state=on]:bg-accent data-[state=on]:text-foreground";

// Adding a source inline: what kind, the field for it, the stance preselected by kind until the person picks one.
export const SourcesProtoAddForm = () => {
  const initial = useSourcesProtoStore((s) => s.adding) ?? "path";
  const setAdding = useSourcesProtoStore((s) => s.setAdding);
  const addSource = useSourcesProtoStore((s) => s.addSource);
  const pasted = initial === "text";
  const form = useForm<AddSourceValues>({
    defaultValues: { kind: initial, path: "", ref: "", label: pasted ? PASTED.label : "", body: pasted ? PASTED.body : "", stance: defaultStance(initial, "") },
  });
  const { kind, stance } = form.watch();

  const restance = (nextKind: SourceKind, path: string) => {
    if (!form.formState.dirtyFields.stance) form.setValue("stance", defaultStance(nextKind, path));
  };

  const submit = form.handleSubmit((v) => {
    const name = { path: v.path.trim(), text: v.label.trim(), doc: v.ref, memory: v.ref, project: v.ref }[v.kind];
    if (name === "") return;
    addSource({ kind: v.kind, name, stance: v.stance });
  });

  return (
    <form onSubmit={submit} aria-label="Add a source" className="animate-in fade-in-0 slide-in-from-top-1 space-y-4 rounded-lg border border-border p-4 duration-200 ease-out">
      <ToggleGroup
        type="single"
        variant="outline"
        size="sm"
        aria-label="Kind of source"
        value={kind}
        onValueChange={(next) => {
          if (!next) return;
          form.setValue("kind", next as SourceKind);
          form.setValue("ref", "");
          restance(next as SourceKind, form.getValues("path"));
        }}
        className="flex-wrap"
      >
        {KINDS.map((k) => (
          <ToggleGroupItem key={k} value={k} className={itemClass}>
            {KIND_LABEL[k]}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      {kind === "path" && (
        <div className="space-y-2">
          <label htmlFor="source-path" className="text-sm font-medium">
            Path
          </label>
          <Input
            id="source-path"
            placeholder="practices/ or docs/standards.md"
            className="font-mono text-[13px]"
            {...form.register("path", { onChange: (e: { target: { value: string } }) => restance("path", e.target.value) })}
          />
          <p className="text-xs text-muted-foreground">A file or folder, relative to the project's checkout.</p>
        </div>
      )}
      {(kind === "doc" || kind === "memory" || kind === "project") && (
        <FormCombobox
          key={kind}
          control={form.control}
          name="ref"
          label={KIND_LABEL[kind]}
          placeholder={`Choose a ${KIND_LABEL[kind].toLowerCase()}`}
          options={PICKABLE[kind].map((v) => ({ value: v, label: v }))}
        />
      )}
      {kind === "text" && <SourcesProtoTextFields form={form} />}
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <span className="text-sm font-medium">Stance</span>
        <SourceStance label="the new source" size="sm" value={stance} onChange={(v) => form.setValue("stance", v, { shouldDirty: true })} />
        <span className="min-w-0 flex-1 basis-48 text-xs text-muted-foreground">{STANCE_HINT[stance]}</span>
      </div>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="ghost" size="sm" onClick={() => setAdding(null)}>
          Cancel
        </Button>
        <Button type="submit" size="sm">
          Add
        </Button>
      </div>
    </form>
  );
};
