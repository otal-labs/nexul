import { Link2, Plus } from "lucide-react";

import { useFormDialog } from "@/hooks/useFormDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { CreateInvitationFormSchema, type CreateInvitationFormData, type CreatedInvitation } from "@/models/Invitation";
import { ActionButton } from "@/components/action/ActionButton";
import { CreateInvitationForm } from "@/components/member/CreateInvitationForm";
import { cn } from "@/lib/utils";

interface CreateInvitationDialogProps {
  onCreated: (invitation: CreatedInvitation) => void;
  // Opened from a role on the Access page: the link starts out giving it, in the workspace in view.
  role?: { id: string; name: string };
  // In a People band under 40rem that also holds the find field, the link mark says it alone; the name stays in its label.
  markAloneWhenNarrow?: boolean;
}

export const CreateInvitationDialog = ({ onCreated, role, markAloneWhenNarrow = false }: CreateInvitationDialogProps) => {
  const { open } = useFormDialog();
  const selectedWorkspaceId = useWorkspaceStore((state) => state.selectedWorkspaceId);

  const create = async () => {
    await open<CreateInvitationFormData>({
      title: "Invite by link",
      description: "For someone without an account. The link works once.",
      schema: CreateInvitationFormSchema,
      okLabel: "Create link",
      form: <CreateInvitationForm onCreated={onCreated} />,
      formOptions: { defaultValues: { expires_in_days: 7, grants: [{ workspace_id: selectedWorkspaceId, role_ids: role ? [role.id] : [] }] } },
      dialogClassName: "sm:max-w-xl",
    });
  };

  return (
    <ActionButton permission="members:write" verb="invite people" icon={role ? Link2 : Plus} type="button" onClick={() => void create()} {...(role && { variant: "outline", size: "sm", "aria-label": `Invite to ${role.name} by link`, title: `Invite to ${role.name} by link` })}>
      {role ? <span className={cn(markAloneWhenNarrow && "@max-[40rem]:sr-only")}>Invite by link</span> : "Invite"}
    </ActionButton>
  );
};
