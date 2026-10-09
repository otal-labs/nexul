import { TemplateOriginLine } from "@/components/templates/TemplateOriginLine";
import { useFetchTemplates } from "@/hooks/TemplateHooks";
import type { Play } from "@/models/Play";
import { findTemplate } from "@/models/Template";

interface PlayTemplateLineProps {
  play: Play;
  canWrite: boolean;
}

// A built-in play's instructions were copied from the instance template when the workspace was made.
export const PlayTemplateLine = ({ play, canWrite }: PlayTemplateLineProps) => {
  const { data: templates } = useFetchTemplates();
  const instance = findTemplate(templates, "play_instructions", play.builtin_key);
  if (!instance) return null;
  return (
    <TemplateOriginLine
      kind="play_instructions"
      templateKey={play.builtin_key}
      at={{ scope: "workspace", workspace_id: play.workspace_id }}
      state={play.instructions === instance.body ? "matches" : "differs"}
      canReset={canWrite}
    />
  );
};
