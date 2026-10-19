import { useNavigate } from "react-router";

import { PlayForm } from "@/components/play/PlayForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { emptyPlayForm, playToFormData, SavePlayFormSchema, type Play, type SavePlayFormData } from "@/models/Play";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// New play and Edit play, each opening the one form; the control that calls them gates itself.
export const usePlayDialogs = () => {
  const { open } = useFormDialog();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);

  const create = async () => {
    const result = await open<SavePlayFormData>({
      title: "New play",
      schema: SavePlayFormSchema,
      okLabel: "Create play",
      form: <PlayForm workspaceId={workspaceId} />,
      formOptions: { defaultValues: emptyPlayForm() },
    });
    const id = (result.data as { id?: string } | null)?.id;
    if (result.success && id) void navigate(wsPath(`/plays/${id}`));
  };

  const edit = (play: Play) =>
    open<SavePlayFormData>({
      title: `Edit ${play.label}`,
      schema: SavePlayFormSchema,
      okLabel: "Save",
      form: <PlayForm workspaceId={workspaceId} editing={play} />,
      formOptions: { defaultValues: playToFormData(play) },
    });

  return { create, edit };
};
