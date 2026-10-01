import { useQueryClient } from "@tanstack/react-query";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { CloneProjectTargetSelect } from "@/components/templates/CloneProjectTargetSelect";
import { CloneWorkspaceTargetSelect } from "@/components/templates/CloneWorkspaceTargetSelect";
import { FormSelect } from "@/components/ticket/FormSelect";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { templateQuery, useCloneTemplate } from "@/hooks/TemplateHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { BELOW_SCOPE, cloneTarget, type CloneSource, type CloneTemplateFormData } from "@/models/Template";

const OVERWRITE_MESSAGE = {
  instance: "The instance template has been edited. Cloning overwrites that text.",
  workspace: "That workspace has its own text for this template. Cloning overwrites it.",
  project: "That project has its own text for this template. Cloning overwrites it.",
};

interface CloneTemplateFormProps {
  source: CloneSource;
}

export const CloneTemplateForm = ({ source }: CloneTemplateFormProps) => {
  const { control, watch, setValue, onSubmit, setStayOpen, onAfterSubmit } = useFormDialogContext<CloneTemplateFormData>();
  const client = useQueryClient();
  const clone = useCloneTemplate();
  const { open: confirm } = useConfirmationDialog();
  const canWriteInstance = useHasInstancePermission("templates:write");
  const below = BELOW_SCOPE[source.kind];
  const scope = watch("scope");

  const scopes = [
    ...(canWriteInstance && source.from.scope !== "instance" ? [{ value: "instance", label: "Instance (make it the default)" }] : []),
    { value: below, label: below === "workspace" ? "A workspace" : "A project" },
  ];

  // A cancelled overwrite keeps the dialog open so another target can be picked.
  onAfterSubmit(() => setStayOpen(false));
  onSubmit(async (input) => {
    const to = cloneTarget(input);
    const current = await client.fetchQuery({ ...templateQuery(source.kind, source.key, to), staleTime: 0 });
    if (current.edited) {
      const ok = await confirm({ title: "Overwrite its text?", message: OVERWRITE_MESSAGE[to.scope], confirmLabel: "Overwrite" });
      if (!ok) {
        setStayOpen(true);
        return input;
      }
    }
    await clone.mutateAsync({ kind: source.kind, key: source.key, from: source.from, to });
    return input;
  });

  return (
    <div className="space-y-4">
      <FormSelect
        control={control}
        name="scope"
        label="Clone to"
        options={scopes}
        onChangeValue={() => setValue("target", "")}
      />
      {scope === "workspace" && <CloneWorkspaceTargetSelect source={source} />}
      {scope === "project" && <CloneProjectTargetSelect source={source} />}
    </div>
  );
};
