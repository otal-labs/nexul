import { useSearchParams } from "react-router";

import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { DEFAULT_SETTINGS_SECTION, isSettingsSection, SettingsNav } from "@/components/settings/SettingsNav";
import { SettingsPageContent } from "@/components/settings/SettingsPageContent";
import { useFetchMe, useFetchSettings } from "@/hooks/AuthHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";

// Same section-per-view shape as ProjectSettingsPage: ?section= drives the card, SettingsNav lists sections.
export const SettingsPage = () => {
  const { data: settings, isPending, error } = useFetchSettings();
  const { data: me } = useFetchMe();
  // Same permission the roles REST endpoints require server-side.
  const canManageRoles = useHasPermission("roles:write");
  const canReadPlays = useHasPermission("plays:read");
  const canWritePlays = useHasPermission("plays:write");
  const canDeletePlays = useHasPermission("plays:delete");
  const canManageMentionLayout = useHasPermission("workspaces:write");
  const canReadMemories = useHasPermission("memories:read");
  const isInstanceAdmin = me?.user?.can_create_workspace ?? false;

  const [searchParams] = useSearchParams();
  const rawSection = searchParams.get("section");
  const section = isSettingsSection(rawSection) ? rawSection : DEFAULT_SETTINGS_SECTION;

  return (
    <Container className="mx-auto max-w-5xl py-10">
      <PageHeader
        className="mb-8"
        eyebrow="Workspace"
        title="Settings"
        subtitle="How this instance connects, who holds keys, and what the board can contain."
      />
      <div className="flex flex-col gap-6 md:flex-row md:items-start md:gap-8">
        <SettingsNav
          active={section}
          showInstanceAccess={isInstanceAdmin}
          showRoles={canManageRoles}
          showPlays={canReadPlays}
          showMentionLayout={canManageMentionLayout}
          showInterviewTemplate={canReadMemories}
        />
        <div className="min-w-0 flex-1 space-y-6">
          {isPending && <LoadingDisplay />}
          {error && <ErrorDisplay error={error} />}
          <SettingsPageContent
            section={section}
            settings={settings}
            isInstanceAdmin={isInstanceAdmin}
            canManageRoles={canManageRoles}
            canReadPlays={canReadPlays}
            canWritePlays={canWritePlays}
            canDeletePlays={canDeletePlays}
            canManageMentionLayout={canManageMentionLayout}
          />
        </div>
      </div>
    </Container>
  );
};
