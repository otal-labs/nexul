import { Microheader, microheaderClass } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ProtoGrant, ProtoInvite } from "@/proto/invitations/data";
import { ExpiryLine, LATER_LINE, ProviderButtons, RoleText, SignedInAs } from "@/proto/invitations/parts";

// Each role is its own card; after sign-in the workspace lists its projects as hairline rows under the cards.
const GrantBlock = ({ grant, signedIn }: { grant: ProtoGrant; signedIn: boolean }) => (
  <section className="space-y-2">
    <div className="flex items-baseline justify-between gap-3">
      <h2 className="min-w-0 text-sm font-semibold break-words">{grant.workspace}</h2>
      <span className="shrink-0 font-mono text-xs text-muted-foreground">
        {grant.roles.length} {grant.roles.length === 1 ? "role" : "roles"}
      </span>
    </div>
    <ul className="space-y-2">
      {grant.roles.map((role) => (
        <li key={role.id} className="rounded-lg border bg-card p-4 shadow-card">
          <RoleText role={role} />
        </li>
      ))}
    </ul>
    {signedIn && (
      <div className="pt-2">
        <Microheader>Projects</Microheader>
        {grant.projects.length === 0 && <p className="py-2 text-xs text-muted-foreground">No projects yet.</p>}
        {grant.projects.length > 0 && (
          <ul className="divide-y divide-border">
            {grant.projects.map((project) => (
              <li key={project} className="py-2 text-[13px]">
                {project}
              </li>
            ))}
          </ul>
        )}
      </div>
    )}
  </section>
);

export const VariantCards = ({ invite, signedIn }: { invite: ProtoInvite; signedIn: boolean }) => (
  <div className="flex min-h-full items-center justify-center px-4 py-10 sm:px-6 sm:py-12">
    <div className="panel w-full max-w-xl p-6 [--panel-opacity:72%] sm:p-8">
      <header className="space-y-2">
        <p className={microheaderClass}>Invitation · {invite.instanceHost}</p>
        <h1 className={cn(displayTitleClass, "text-[2rem]")}>You're invited to {invite.instanceName}</h1>
        <p className="text-sm text-muted-foreground">
          {signedIn ? "Here's what accepting gives you." : "Sign in to join. Here's what you'll get."}
        </p>
      </header>
      <div className="mt-6 space-y-6">
        {invite.grants.map((grant) => (
          <GrantBlock key={grant.workspace} grant={grant} signedIn={signedIn} />
        ))}
      </div>
      {signedIn && <p className="mt-4 text-xs text-muted-foreground">{LATER_LINE}</p>}
      {!signedIn && <ProviderButtons className="mt-6" />}
      {signedIn && (
        <div className="mt-6 space-y-3">
          <SignedInAs invite={invite} />
          <Button type="button" className="w-full">
            Accept invitation
          </Button>
          <Button type="button" variant="ghost" className="w-full">
            Decline
          </Button>
        </div>
      )}
      <ExpiryLine invite={invite} className="mt-4" />
    </div>
  </div>
);
