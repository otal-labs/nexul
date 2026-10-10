import { Plus } from "lucide-react";

import { useFormDialog } from "@/hooks/useFormDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { CreateInvitationFormSchema, type CreateInvitationFormData, type CreatedInvitation } from "@/models/Invitation";
import { EveryProject } from "@/models/Team";
import { Button } from "@/components/ui/button";
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
      description: "Pick the workspaces and roles the link grants. It works once.",
      schema: CreateInvitationFormSchema,
      okLabel: "Create link",
      form: <CreateInvitationForm onCreated={onCreated} />,
      formOptions: { defaultValues: { expires_in_days: 7, grants: [{ workspace_id: selectedWorkspaceId, role_id: "", allow: [], deny: [], every_project: EveryProject.Role, project_access: [] }] } },
      dialogClassName: "max-w-2xl",
    });
  };

  return <Button type="button" onClick={() => void create()}><Plus className="size-4" />Invite</Button>;
};
