import { Navigate, useLocation, useParams } from "react-router";

import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { YourSettingsContent } from "@/components/you/YourSettingsContent";
import {
  DEFAULT_YOUR_SETTINGS_SECTION,
  isYourSettingsSection,
  YourSettingsNav,
} from "@/components/you/YourSettingsNav";
import { movedSettingsTarget } from "@/utils/SettingsRedirects";

export const YourSettingsPage = () => {
  const { section: rawSection } = useParams();
  const { search, hash } = useLocation();
  const moved = movedSettingsTarget(rawSection, search, hash);
  const section = isYourSettingsSection(rawSection) ? rawSection : DEFAULT_YOUR_SETTINGS_SECTION;

  return (
    <>
      {moved && <Navigate to={moved} replace />}
      {!moved && (
        <Container className="mx-auto max-w-5xl py-10">
          <PageHeader
            className="mb-8"
            eyebrow="You"
            title="Settings"
            subtitle="Your profile, how Nexul looks, and where you're signed in."
          />
          <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8">
            <YourSettingsNav active={section} />
            <div className="min-w-0 flex-1 space-y-6">
              <YourSettingsContent section={section} />
            </div>
          </div>
        </Container>
      )}
    </>
  );
};
