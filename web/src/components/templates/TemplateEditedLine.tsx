import { personLabel } from "@nexul/client-core/person";

import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { usePerson } from "@/hooks/PeopleHooks";
import type { Template } from "@/models/Template";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface TemplateEditedLineProps {
  template: Template;
}

// State by exception: an edited template says so, one that still reads as shipped says nothing.
export const TemplateEditedLine = ({ template }: TemplateEditedLineProps) => {
  const editor = usePerson(template.updated_by ?? "");
  const detail = [template.updated_by && `by ${personLabel(editor)}`, template.updated_at && formatRelativeTime(template.updated_at)]
    .filter(Boolean)
    .join(" ");
  if (!template.edited) return null;
  return (
    <p className="flex min-w-0">
      <SettingsStatus tone="info" detail={detail || undefined}>
        {template.added ? "Added" : "Edited"}
      </SettingsStatus>
    </p>
  );
};
