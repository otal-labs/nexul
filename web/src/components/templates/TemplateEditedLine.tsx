import { personLabel } from "@nexul/client-core/person";

import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { usePerson } from "@/hooks/PeopleHooks";
import type { Template } from "@/models/Template";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface TemplateEditedLineProps {
  template: Template;
}

export const TemplateEditedLine = ({ template }: TemplateEditedLineProps) => {
  const editor = usePerson(template.updated_by ?? "");
  const detail = [template.updated_by && `by ${personLabel(editor)}`, template.updated_at && formatRelativeTime(template.updated_at)]
    .filter(Boolean)
    .join(" ");
  return (
    <p className="flex min-w-0">
      {template.edited && (
        <SettingsStatus tone="info" detail={detail || undefined}>
          Edited
        </SettingsStatus>
      )}
      {!template.edited && <SettingsStatus tone="muted">Default</SettingsStatus>}
    </p>
  );
};
