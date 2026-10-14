import { Plus } from "lucide-react";

import { useFormDialog } from "@/hooks/useFormDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { CreateInvitationFormSchema, type CreateInvitationFormData, type CreatedInvitation } from "@/models/Invitation";
import { ActionButton } from "@/components/action/ActionButton";
import { CreateInvitationForm } from "@/components/member/CreateInvitationForm";

interface CreateInvitationDialogProps {
  onCreated: (invitation: CreatedInvitation) => void;
}

export const CreateInvitationDialog = ({ onCreated }: CreateInvitationDialogProps) => {
  const { open } = useFormDialog();
  const selectedWorkspaceId = useWorkspaceStore((state) => state.selectedWorkspaceId);

  const create = async () => {
    await open<CreateInvitationFormData>({
      title: "Create invitation link",
      description: "For someone without an account here. Pick the workspaces and the roles the link gives, then copy it and send it yourself. It works once.",
      schema: CreateInvitationFormSchema,
      okLabel: "Create link",
      form: <CreateInvitationForm onCreated={onCreated} />,
      formOptions: { defaultValues: { expires_in_days: 7, grants: [{ workspace_id: selectedWorkspaceId, role_ids: [] }] } },
      dialogClassName: "max-w-2xl",
    });
  };

  return (
    <ActionButton permission="members:write" verb="invite people" icon={Plus} type="button" onClick={() => void create()}>
      Invite
    </ActionButton>
  );
};
