import { HeaderFacts } from "@/components/HeaderFacts";
import type { SettingsSection } from "@/components/settings/SettingsNav";
import type { YourSettingsSection } from "@/components/you/YourSettingsNav";
import { useFetchMe, useFetchSettings, useListPATs, useListSessions } from "@/hooks/AuthHooks";
import { useFetchConnectors } from "@/hooks/ConnectorsHooks";
import { useFetchExposures, useFetchGateways } from "@/hooks/DnsHooks";
import { useFetchInvitations } from "@/hooks/InvitationHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useFetchTemplates } from "@/hooks/TemplateHooks";
import { getThemeDefinition } from "@/lib/themePalettes";
import { useThemeStore } from "@/stores/themeStore";
import { formatShortDate } from "@/utils/TimeUtility";
import { loginHandle } from "@/models/User";

const count = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;


const ProfileMeta = () => {
  const { data: me } = useFetchMe();
  return (
    <HeaderFacts
      parts={[me && <span className="font-mono">{loginHandle(me.user.login)}</span>, me && `Joined ${formatShortDate(me.user.created_at)}`]}
    />
  );
};

const AppearanceMeta = () => {
  const themeId = useThemeStore((s) => s.themeId);
  const mode = useThemeStore((s) => s.appearanceMode);
  const theme = useThemeStore((s) => s.theme);
  const modeText: Record<typeof mode, string> = { system: `System, ${theme}`, dark: "Dark", light: "Light" };
  return <HeaderFacts parts={[`${getThemeDefinition(themeId).label} theme`, modeText[mode]]} />;
};

const SecurityMeta = () => {
  const { data: sessions } = useListSessions();
  const { data: pats } = useListPATs();
  const active = pats?.tokens.filter((token) => !token.revoked_at).length;
  return (
    <HeaderFacts
      parts={[
        sessions && `${count(sessions.sessions.length, "device")} signed in`,
        active !== undefined && count(active, "active token"),
      ]}
    />
  );
};

const InstanceMeta = () => {
  const { data: settings } = useFetchSettings();
  return <HeaderFacts parts={["Instance-wide", settings && <span className="font-mono">{settings.instance_url}</span>]} />;
};

const TeamMeta = () => {
  const { data: team } = useFetchTeam();
  const { data: invitations } = useFetchInvitations();
  return (
    <HeaderFacts
      parts={[
        "Instance-wide",
        team && count(team.people.length, "person", "people"),
        invitations && count(invitations.invitations.length, "open invitation link"),
      ]}
    />
  );
};

const SignInMeta = () => {
  const { data: settings } = useFetchSettings();
  const state = (on: boolean | undefined) => (on ? "on" : "off");
  return (
    <HeaderFacts
      parts={[
        "Instance-wide",
        settings && `Discord ${state(settings.discord_oauth_configured)}`,
        settings && `Google ${state(settings.google_oauth_configured)}`,
      ]}
    />
  );
};

const ConnectorsMeta = () => {
  const { data } = useFetchConnectors();
  const connected = data?.filter((entry) => entry.status.configured).length;
  return <HeaderFacts parts={["Instance-wide", data && `${connected} of ${data.length} connected`]} />;
};

const DnsMeta = () => {
  const { data: gateways } = useFetchGateways();
  const { data: exposures } = useFetchExposures();
  return (
    <HeaderFacts
      parts={["Instance-wide", gateways && count(gateways.length, "gateway"), exposures && count(exposures.length, "hostname")]}
    />
  );
};

const TemplatesMeta = () => {
  const { data } = useFetchTemplates();
  const edited = data?.filter((template) => template.edited).length;
  return <HeaderFacts parts={["Instance-wide", data && `${count(data.length, "template")}, ${edited} edited`]} />;
};

interface SettingsHeaderMetaProps {
  section: YourSettingsSection | SettingsSection;
}

export const SettingsHeaderMeta = ({ section }: SettingsHeaderMetaProps) => (
  <>
    {section === "profile" && <ProfileMeta />}
    {section === "appearance" && <AppearanceMeta />}
    {section === "security" && <SecurityMeta />}
    {section === "instance" && <InstanceMeta />}
    {section === "team" && <TeamMeta />}
    {section === "sign-in" && <SignInMeta />}
    {section === "connectors" && <ConnectorsMeta />}
    {section === "dns" && <DnsMeta />}
    {section === "templates" && <TemplatesMeta />}
  </>
);
