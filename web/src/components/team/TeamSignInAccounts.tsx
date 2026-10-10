import { ProviderMark } from "@/components/ProviderMarks";
import type { TeamPerson } from "@/models/Team";
import { providerLabel } from "@/models/User";

interface TeamSignInAccountsProps {
  person: TeamPerson;
}

// A provider this build has no mark for (the dev login) is left out rather than drawn blank; with no username at all the login stands in.
export const TeamSignInAccounts = ({ person }: TeamSignInAccountsProps) => {
  const known = person.providers.filter((provider) => provider in providerLabel);
  const named = known.some((provider) => person.usernames?.[provider]);
  return (
    <>
      {known.map((provider) => (
        <span key={provider} className="inline-flex shrink-0 items-center gap-1 [&_svg]:size-3">
          <span role="img" aria-label={providerLabel[provider]} title={providerLabel[provider]} className="inline-flex">
            <ProviderMark provider={provider} />
          </span>
          {person.usernames?.[provider] && (
            <span className="font-mono">
              {provider === "github" && "@"}
              {person.usernames[provider]}
            </span>
          )}
          <span aria-hidden>·</span>
        </span>
      ))}
      {!named && <span className="shrink-0 font-mono">@{person.login} ·</span>}
    </>
  );
};
