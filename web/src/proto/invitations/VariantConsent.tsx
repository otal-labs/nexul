import { microheaderClass } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ProtoGrant, ProtoInvite } from "@/proto/invitations/data";
import { ExpiryLine, LATER_LINE, MarkPair, ProjectsLine, ProviderButtons, RoleText, SignedInAs } from "@/proto/invitations/parts";

const WorkspaceTile = ({ name }: { name: string }) => (
  <span className="grid size-5 shrink-0 place-items-center rounded-[5px] bg-surface-2 font-mono text-[10px] font-semibold text-muted-foreground ring-1 ring-border">
    {name[0]}
  </span>
);

// One workspace: its name, then a card of the roles given there, each name over its one line; after sign-in the card's foot names the projects.
const GrantCard = ({ grant, signedIn }: { grant: ProtoGrant; signedIn: boolean }) => (
  <section className="space-y-2">
    <h2 className="flex min-w-0 items-center gap-2 text-[13px] font-medium">
      <WorkspaceTile name={grant.workspace} />
      <span className="min-w-0 break-words">{grant.workspace}</span>
    </h2>
    <div className="overflow-hidden rounded-lg border bg-card shadow-card">
      <ul className="divide-y divide-border">
        {grant.roles.map((role) => (
          <li key={role.id} className="px-3.5 py-3">
            <RoleText role={role} />
          </li>
        ))}
      </ul>
      {signedIn && <ProjectsLine projects={grant.projects} className="border-t bg-surface-2/60 px-3.5 py-2.5" />}
    </div>
  </section>
);

export const VariantConsent = ({ invite, signedIn }: { invite: ProtoInvite; signedIn: boolean }) => (
  <div className="flex min-h-full items-center justify-center px-4 py-10 sm:px-6 sm:py-12">
    <div className="panel w-full max-w-[28rem] p-6 [--panel-opacity:72%] sm:p-8">
      <header className="space-y-4 text-center">
        <MarkPair person={signedIn ? invite.person.name : undefined} />
        <div className="space-y-1.5">
          <p className={microheaderClass}>Invitation</p>
          <h1 className={cn(displayTitleClass, "text-[1.75rem]")}>Join {invite.instanceName}</h1>
          <p className="font-mono text-xs break-all text-muted-foreground">{invite.instanceHost}</p>
        </div>
      </header>
      <div className="mt-7 space-y-5">
        {invite.grants.map((grant) => (
          <GrantCard key={grant.workspace} grant={grant} signedIn={signedIn} />
        ))}
      </div>
      {signedIn && <p className="mt-3 text-xs text-muted-foreground">{LATER_LINE}</p>}
      {!signedIn && (
        <div className="mt-7 space-y-3">
          <p className="text-center text-[13px] text-muted-foreground">Sign in to accept</p>
          <ProviderButtons />
        </div>
      )}
      {signedIn && (
        <div className="mt-7 space-y-3">
          <SignedInAs invite={invite} />
          <div className="grid grid-cols-2 gap-2">
            <Button type="button" variant="outline">
              Decline
            </Button>
            <Button type="button">Accept invitation</Button>
          </div>
        </div>
      )}
      <ExpiryLine invite={invite} className="mt-5 text-center" />
    </div>
  </div>
);
