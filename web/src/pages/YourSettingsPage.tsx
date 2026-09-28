import { useSearchParams } from "react-router";

import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { YourSettingsContent } from "@/components/you/YourSettingsContent";
import { isYourSettingsSection, YourSettingsNav } from "@/components/you/YourSettingsNav";

export const YourSettingsPage = () => {
  const [searchParams] = useSearchParams();
  const raw = searchParams.get("section");
  const section = isYourSettingsSection(raw) ? raw : "profile";

  return (
    <Container className="mx-auto max-w-5xl py-10">
      <PageHeader className="mb-8" eyebrow="You" title="Settings" subtitle="Your profile, how Nexul looks, and where you're signed in." />
      <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8">
        <YourSettingsNav active={section} />
        <div className="min-w-0 flex-1 space-y-6">
          <YourSettingsContent section={section} />
        </div>
      </div>
    </Container>
  );
};
