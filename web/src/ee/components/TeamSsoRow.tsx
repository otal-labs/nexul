import { ArrowRightLeft } from "lucide-react";

import { Microheader } from "@/components/Microheader";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useMoveSsoSignIn, usePublicSso } from "@/ee/hooks/SsoHooks";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFetchTeam } from "@/hooks/TeamHooks";
import type { TeamPerson } from "@/models/Team";
import { loginHandle } from "@/models/User";
import { personName } from "@/utils/TeamUtility";

const SSO_PROVIDER = "oidc";

// The Team sends the single sign-on identity among the providers, though Provider names only the ones with a mark.
const providersOf = (person: TeamPerson): readonly string[] => person.providers;

// An account a first sign-in through the provider made: single sign-on is its only way in.
const onlySso = (person: TeamPerson): boolean => providersOf(person).length === 1 && providersOf(person)[0] === SSO_PROVIDER;

// Whether the person signs in through the provider; for the instance Owner, also the way to move that sign-in here
// from the duplicate account a first sign-in made.
export const TeamSsoRow = ({ person }: { person: TeamPerson }) => {
  const { data: sso } = usePublicSso();
  const { data: team } = useFetchTeam();
  const canMove = useHasInstancePermission("instance:write");
  const move = useMoveSsoSignIn();
  const { open: confirm } = useConfirmationDialog();
  if (!sso?.active) return null;
  const linked = providersOf(person).includes(SSO_PROVIDER);
  const duplicates = (team?.people ?? []).filter((other) => other.id !== person.id && other.status !== "removed" && onlySso(other));
  if (!linked && (!canMove || duplicates.length === 0)) return null;
  const name = personName(person);

  const moveFrom = async (duplicate: TeamPerson) => {
    const ok = await confirm({
      title: `Move ${loginHandle(duplicate.login)} to ${name}?`,
      message: `Signing in through ${sso.name} as ${duplicate.login} opens ${name}'s account from now on. The other account is disabled and signed out, and what it made stays.`,
      confirmLabel: "Move sign-in",
    });
    if (ok) move.mutate({ from: duplicate.id, to: person.id });
  };

  return (
    <section aria-labelledby="team-sso-title" className="flex flex-wrap items-center justify-between gap-2 px-6 pb-4">
      <Microheader id="team-sso-title">Single sign-on</Microheader>
      {linked && <span className="text-sm text-muted-foreground">Signs in through {sso.name}</span>}
      {!linked && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="sm" loading={move.isPending}>
              <ArrowRightLeft aria-hidden />
              Move from another account
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="min-w-56">
            {duplicates.map((duplicate) => (
              <DropdownMenuItem key={duplicate.id} onSelect={() => void moveFrom(duplicate)}>
                <span className="min-w-0 truncate">{personName(duplicate)}</span>
                <span className="ml-auto truncate font-mono text-xs text-muted-foreground">{duplicate.login}</span>
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </section>
  );
};
