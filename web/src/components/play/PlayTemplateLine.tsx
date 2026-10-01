import { TemplateOriginLine } from "@/components/templates/TemplateOriginLine";
import { Button } from "@/components/ui/button";
import { useFetchTemplates } from "@/hooks/TemplateHooks";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import type { Play } from "@/models/Play";
import { findTemplate } from "@/models/Template";

interface PlayTemplateLineProps {
  play: Play;
  canWrite: boolean;
}

// A built-in play's instructions were copied from the instance template when the workspace was made.
export const PlayTemplateLine = ({ play, canWrite }: PlayTemplateLineProps) => {
  const { data: templates } = useFetchTemplates();
  const openClone = useCloneTemplateDialog();
  const instance = findTemplate(templates, "play_instructions", play.builtin_key);
  const at = { scope: "workspace" as const, workspace_id: play.workspace_id };

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
      {instance && (
        <TemplateOriginLine
          kind="play_instructions"
          templateKey={play.builtin_key}
          at={at}
          state={play.instructions === instance.body ? "matches" : "differs"}
          canReset={canWrite}
        />
      )}
      <Button
        type="button"
        variant="link"
        size="sm"
        className="h-auto p-0 text-xs"
        aria-label={`Clone ${play.label} instructions to…`}
        onClick={() => void openClone({ kind: "play_instructions", key: play.builtin_key, name: play.label, from: at })}
      >
        Clone to…
      </Button>
    </div>
  );
};
