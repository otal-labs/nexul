import { usePerson } from "@/hooks/PeopleHooks";
import { personLabel } from "@/models/Person";
import type { Template } from "@/models/Template";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface TemplateEditedLineProps {
  template: Template;
}

export const TemplateEditedLine = ({ template }: TemplateEditedLineProps) => {
  const editor = usePerson(template.updated_by ?? "");
  const parts = template.edited
    ? ["Edited", ...(template.updated_by ? [`by ${personLabel(editor)}`] : []), ...(template.updated_at ? [formatRelativeTime(template.updated_at)] : [])]
    : ["Default"];
  return <p className="text-xs text-muted-foreground">{parts.join(" · ")}</p>;
};
