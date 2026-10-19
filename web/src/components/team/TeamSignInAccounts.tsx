import { SignInAccount } from "@/components/SignInAccounts";
import type { TeamPerson } from "@/models/Team";
import { knownProviders, loginHandle } from "@/models/User";

interface TeamSignInAccountsProps {
  person: TeamPerson;
}

// With no username at all the login stands in.
export const TeamSignInAccounts = ({ person }: TeamSignInAccountsProps) => {
  const known = knownProviders(person.providers);
  const named = known.some((provider) => person.usernames?.[provider]);
  return (
    <>
      {known.map((provider) => (
        <span key={provider} className="inline-flex shrink-0 items-center gap-1">
          <SignInAccount provider={provider} username={person.usernames?.[provider]} />
          <span aria-hidden>·</span>
        </span>
      ))}
      {!named && <span className="shrink-0 font-mono">{loginHandle(person.login)} ·</span>}
    </>
  );
};
