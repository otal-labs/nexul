import { useFormContext, useWatch } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { normalizeSlugInput, slugify, type WorkspaceGeneralFormData } from "@/models/Workspace";

interface WorkspaceUrlFieldProps {
  savedSlug: string;
}

// The workspace's name in every link, typed after this instance's own address.
export const WorkspaceUrlField = ({ savedSlug }: WorkspaceUrlFieldProps) => {
  const { control, getValues, setValue } = useFormContext<WorkspaceGeneralFormData>();
  const slug = useWatch({ control, name: "slug" });
  const name = useWatch({ control, name: "name" });
  const suggestion = slugify(name);

  const suggest = () => setValue("slug", slugify(getValues("name")), { shouldDirty: true, shouldValidate: true });

  return (
    <div className="space-y-2">
      <FormInput
        control={control}
        name="slug"
        id="workspace-slug"
        label="URL"
        leading={`${window.location.host}/`}
        transform={normalizeSlugInput}
        spellCheck={false}
        autoComplete="off"
        className="max-w-xs font-mono"
      />
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
        {slug !== savedSlug && (
          <p className="text-xs text-muted-foreground">Links using the old address will stop working.</p>
        )}
        {suggestion !== slug && (
          <Button type="button" variant="link" size="sm" className="h-auto p-0 text-xs" onClick={suggest}>
            Suggest from name
          </Button>
        )}
      </div>
    </div>
  );
};
