import { useState } from "react";
import { Check, Copy, Plus } from "lucide-react";

import { useFormDialog } from "@/hooks/useFormDialog";
import { CreateInvitationFormSchema, type CreateInvitationFormData, type CreatedInvitation } from "@/models/Invitation";
import { Button } from "@/components/ui/button";
import { CreateInvitationForm } from "@/components/member/CreateInvitationForm";

export const CreateInvitationDialog = () => {
  const { open } = useFormDialog();
  const [created, setCreated] = useState<CreatedInvitation | null>(null);
  const [copied, setCopied] = useState(false);

  const create = async () => {
    setCreated(null);
    setCopied(false);
    await open<CreateInvitationFormData>({
      title: "Create invitation link",
      description: "Choose the workspaces and roles this one-use link grants.",
      schema: CreateInvitationFormSchema,
      okLabel: "Create link",
      form: <CreateInvitationForm onCreated={setCreated} />,
      formOptions: { defaultValues: { expires_in_days: 7, grants: [{ workspace_id: "", role_id: "", allow: [], deny: [] }] } },
      dialogClassName: "max-w-2xl",
    });
  };

  const copy = async () => {
    if (!created) return;
    try {
      await navigator.clipboard.writeText(created.url);
      setCopied(true);
    } catch {
      // The link remains visible for manual copying.
    }
  };

  return (
    <>
      <Button type="button" onClick={() => void create()}><Plus className="size-4" />Invite</Button>
      {created && (
        <div className="mt-4 space-y-3 rounded-md border bg-muted/30 p-4" role="status">
          <p className="text-sm font-medium">Copy this invitation now — it won&apos;t be shown again.</p>
          <p className="break-all rounded-md bg-card p-2 font-mono text-xs">{created.url}</p>
          <Button type="button" variant="outline" size="sm" onClick={() => void copy()}>{copied ? <Check className="size-4" /> : <Copy className="size-4" />}{copied ? "Copied" : "Copy link"}</Button>
        </div>
      )}
    </>
  );
};
