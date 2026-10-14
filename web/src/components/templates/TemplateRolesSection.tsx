import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RoleEditorDialog } from "@/components/settings/RoleEditorDialog";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { TemplateRoleRow } from "@/components/templates/TemplateRoleRow";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchTemplates } from "@/hooks/TemplateHooks";
import { useRowGlide } from "@/hooks/useRowGlide";
import { ROLE_TEMPLATE } from "@/models/Template";
import { templateRole } from "@/models/TemplateRole";

// The template roles every new workspace copies beside its Owner; the Settings section opens only on templates:write.
export const TemplateRolesSection = () => {
  const { data: templates, isPending: templatesPending, error: templatesError } = useFetchTemplates();
  const { data: catalog, isPending: catalogPending, error: catalogError } = useFetchPermissionCatalog();
  const [creating, setCreating] = useState(false);
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();

  const isPending = templatesPending || catalogPending;
  const error = templatesError ?? catalogError;
  const roles = templates?.filter((template) => template.kind === ROLE_TEMPLATE);

  return (
    <SettingsCard
      id="template-roles"
      title="Roles"
      description="Every new workspace starts with a copy of each beside its Owner. Editing one later changes only workspaces created after."
      footer={
        roles &&
        catalog && (
          <Button variant="outline" size="sm" onClick={() => setCreating(true)}>
            <PlusIcon className="size-4" />
            New template role
          </Button>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {roles && catalog && (
        <div ref={glideRef}>
          <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
            {roles.map((template) => (
              <TemplateRoleRow key={template.key} template={template} catalog={catalog} onLeave={prepareGlide} />
            ))}
          </EnterList>
          <RoleEditorDialog open={creating} template={{ startFrom: roles.map(templateRole) }} onClose={() => setCreating(false)} />
        </div>
      )}
    </SettingsCard>
  );
};
