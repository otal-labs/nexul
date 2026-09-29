import { Navigate, useLocation, useParams } from "react-router";

import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { isSettingsSection, SettingsNav, visibleSettingsSections } from "@/components/settings/SettingsNav";
import { SettingsPageContent } from "@/components/settings/SettingsPageContent";
import { useFetchMe, useFetchSettings } from "@/hooks/AuthHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { legacyConfigurationTarget } from "@/utils/SettingsRedirects";

// Same section-per-view shape as ProjectSettingsPage: the :section path segment drives the card, SettingsNav lists sections.
export const ConfigurationPage = () => {
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
  const canManageMembers = useHasPermission("members:write");
  // Members may be managed in a workspace other than the selected one; the scoped Team read answers that.
  const { data: team } = useFetchTeam(!!me && !isInstanceAdmin && !canManageMembers);

  const sections = visibleSettingsSections({
    isInstanceAdmin,
    showRoles: canManageRoles,
    showPlays: canReadPlays,
    showInterviewTemplate: canReadMemories,
    showMentionLayout: canManageMentionLayout,
    showTeam: canManageMembers || !!team,
  });

  const { section: rawSection } = useParams();
  const { search, hash } = useLocation();
  const moved = legacyConfigurationTarget(rawSection, search, hash);
  // A section the viewer can't open (unknown, or gated away) falls back to the first they can; Danger zone is never gated, so it's the last resort.
  const fallback = sections.find((candidate) => candidate !== "danger") ?? "danger";
  const section = isSettingsSection(rawSection) && sections.includes(rawSection) ? rawSection : fallback;

  return (
    <Container className="mx-auto max-w-5xl py-10">
      {moved && <Navigate to={moved} replace />}
      <PageHeader
        className="mb-8"
        eyebrow="Workspace"
        title="Configuration"
        subtitle="What this workspace can contain and who holds keys, and how the whole instance connects."
      />
      <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8">
        <SettingsNav active={section} sections={sections} isInstanceAdmin={isInstanceAdmin} />
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
