import { ProviderMark } from "@/components/ProviderMarks";
import { knownProviders, providerLabel, type Provider } from "@/models/User";

interface SignInAccountProps {
  provider: Provider;
  username: string | undefined;
}

// A sign-in account as others know it: the provider's mark, then the username when it has one (Google has none).
export const SignInAccount = ({ provider, username }: SignInAccountProps) => (
  <span className="inline-flex shrink-0 items-center gap-1 [&_svg]:size-3">
    <span role="img" aria-label={providerLabel[provider]} title={providerLabel[provider]} className="inline-flex">
      <ProviderMark provider={provider} />
    </span>
    {username && (
      <span className="font-mono">
        {provider === "github" && "@"}
        {username}
      </span>
    )}
  </span>
);

interface SignInAccountsProps {
  providers: readonly string[] | undefined;
  usernames: Partial<Record<string, string>> | undefined;
}

// Someone's sign-in accounts without their login, which is the email of a Google or Discord account.
export const SignInAccounts = ({ providers, usernames }: SignInAccountsProps) => (
  <span className="inline-flex min-w-0 items-center gap-2">
    {knownProviders(providers).map((provider) => (
      <SignInAccount key={provider} provider={provider} username={usernames?.[provider]} />
    ))}
  </span>
);
