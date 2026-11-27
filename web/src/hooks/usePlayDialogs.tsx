import { useNavigate } from "react-router";

import { PlayForm } from "@/components/play/PlayForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { agentPlayPath } from "@/models/Agent";
import { emptyPlayForm, playToFormData, SavePlayFormSchema, type Play, type PlayType, type SavePlayFormData } from "@/models/Play";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// New play, Clone and Edit play, each opening the one form; the control that calls them gates itself.
export const usePlayDialogs = () => {
  const { open } = useFormDialog();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);

  const createFrom = async (title: string, defaultValues: SavePlayFormData) => {
    const result = await open<SavePlayFormData>({
      title,
      schema: SavePlayFormSchema,
      okLabel: "Create play",
      form: <PlayForm workspaceId={workspaceId} />,
      formOptions: { defaultValues },
    });
    const id = (result.data as { id?: string } | null)?.id;
    if (result.success && id) void navigate(wsPath(agentPlayPath(defaultValues.type, id)));
  };

  const create = (type: PlayType = "ticket") =>
    createFrom("New play", { ...emptyPlayForm(), type, show_when_stage: type === "ticket" ? "progress" : "" });

  const duplicate = (play: Play) => createFrom(`Clone ${play.label}`, { ...playToFormData(play), label: `${play.label} copy` });

  const edit = (play: Play) =>
    open<SavePlayFormData>({
      title: `Edit ${play.label}`,
      schema: SavePlayFormSchema,
      okLabel: "Save",
      form: <PlayForm workspaceId={workspaceId} editing={play} />,
      formOptions: { defaultValues: playToFormData(play) },
    });

  return { create, duplicate, edit };
};
