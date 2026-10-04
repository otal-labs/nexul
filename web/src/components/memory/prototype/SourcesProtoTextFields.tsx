import { useState, type DragEvent } from "react";
import { Upload } from "lucide-react";
import type { UseFormReturn } from "react-hook-form";

import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { AddSourceValues } from "@/components/memory/prototype/SourcesProtoAddForm";

const readable = (file: File) => file.type.startsWith("text/") || /\.(md|markdown|txt)$/i.test(file.name);

// Pasted text: a label, the text, and a text or markdown file read in the browser as if it had been pasted.
export const SourcesProtoTextFields = ({ form }: { form: UseFormReturn<AddSourceValues> }) => {
  const [refused, setRefused] = useState("");

  const take = async (file: File | undefined) => {
    if (!file) return;
    if (!readable(file)) {
      setRefused(`${file.name} isn't a text or markdown file; paste its text instead.`);
      return;
    }
    setRefused("");
    form.setValue("body", await file.text());
    if (form.getValues("label").trim() === "") form.setValue("label", file.name);
  };

  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    void take(e.dataTransfer.files[0]);
  };

  return (
    <div className="space-y-3" onDragOver={(e) => e.preventDefault()} onDrop={onDrop}>
      <div className="space-y-2">
        <label htmlFor="source-label" className="text-sm font-medium">
          Label
        </label>
        <Input id="source-label" placeholder="Phase 1 handover notes" {...form.register("label")} />
      </div>
      <div className="space-y-2">
        <label htmlFor="source-body" className="text-sm font-medium">
          Text
        </label>
        <Textarea id="source-body" rows={5} className="field-sizing-content max-h-64 min-h-24 resize-none" {...form.register("body")} />
        <label className="flex cursor-pointer items-center gap-2 text-xs text-muted-foreground hover:text-foreground">
          <Upload className="size-3.5" aria-hidden />
          or drop a text or markdown file
          <input type="file" accept=".md,.markdown,.txt,text/*" className="sr-only" onChange={(e) => void take(e.target.files?.[0])} />
        </label>
        {refused !== "" && <p className="text-xs text-destructive">{refused}</p>}
      </div>
    </div>
  );
};
