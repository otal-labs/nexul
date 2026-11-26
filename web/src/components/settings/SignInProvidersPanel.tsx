import { OAuthProviderSection } from "@/components/settings/OAuthProviderSection";
import type { InstanceSettings } from "@/models/User";

// One card per provider, each with its own state and actions; two short cards needed no tab row above them.
export const SignInProvidersPanel = ({ settings }: { settings: InstanceSettings }) => (
  <>
    <OAuthProviderSection provider="discord" settings={settings} />
    <OAuthProviderSection provider="google" settings={settings} />
  </>
);
