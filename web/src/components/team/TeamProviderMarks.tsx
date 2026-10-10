import { ProviderMark } from "@/components/ProviderMarks";
import { providerLabel, type Provider } from "@/models/User";

interface TeamProviderMarksProps {
  providers: Provider[];
}

// A provider this build has no mark for (the dev login) is left out rather than drawn blank.
export const TeamProviderMarks = ({ providers }: TeamProviderMarksProps) => {
  const known = providers.filter((provider) => provider in providerLabel);
  if (known.length === 0) return null;
  const label = `Signs in with ${known.map((provider) => providerLabel[provider]).join(" and ")}`;
  return (
    <span role="img" aria-label={label} title={label} className="inline-flex shrink-0 items-center gap-1 [&_svg]:size-3">
      {known.map((provider) => (
        <ProviderMark key={provider} provider={provider} />
      ))}
    </span>
  );
};
