import { useState } from "react";
import { ArrowLeftRight, FolderKanban, UserRound } from "lucide-react";

import { Logo } from "@/components/Logo";
import { DiscordMark, GithubMark, GoogleMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import { avatarGradient } from "@/lib/avatarGradient";
import { cn } from "@/lib/utils";
import type { ProtoInvite, ProtoRole } from "@/proto/invitations/data";

export const initials = (name: string) =>
  name
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase();

export const PersonAvatar = ({ name, className }: { name: string; className?: string }) => (
  <span
    className={cn("grid size-10 shrink-0 place-items-center rounded-full text-[13px] font-semibold text-white", className)}
    style={{ background: avatarGradient(name) }}
    aria-hidden
  >
    {initials(name)}
  </span>
);

// Before sign-in the second mark is an empty seat: the page says "this place and you" before any text.
export const MarkPair = ({ person }: { person?: string | undefined }) => (
  <div className="flex items-center justify-center gap-3">
    <Logo className="size-10 rounded-[10px] [&_svg]:size-6" />
    <ArrowLeftRight aria-hidden className="size-4 text-muted-foreground" />
    {person && <PersonAvatar name={person} />}
    {!person && (
      <span className="grid size-10 place-items-center rounded-full border border-dashed border-foreground/25 text-muted-foreground" aria-hidden>
        <UserRound className="size-4" />
      </span>
    )}
  </div>
);

export const RoleText = ({ role, className }: { role: ProtoRole; className?: string }) => (
  <div className={cn("min-w-0", className)}>
    <p className="text-sm font-medium break-words">{role.name}</p>
    {role.description && <p className="mt-0.5 text-[13px] leading-5 text-muted-foreground">{role.description}</p>}
  </div>
);

const PROJECTS_SHOWN = 8;

// Names past the eighth fold behind "and N more", so a workspace with fifty projects stays one short paragraph.
export const ProjectsLine = ({ projects, className }: { projects: string[]; className?: string }) => {
  const [open, setOpen] = useState(false);
  const shown = open ? projects : projects.slice(0, PROJECTS_SHOWN);
  const hidden = projects.length - shown.length;
  return (
    <p className={cn("flex items-start gap-2 text-xs leading-5 text-muted-foreground", className)}>
      <FolderKanban aria-hidden className="mt-0.5 size-3.5 shrink-0" />
      {projects.length === 0 && <span>No projects yet</span>}
      {projects.length > 0 && (
        <span>
          <span className="text-foreground/85">
            {projects.length} {projects.length === 1 ? "project" : "projects"}:
          </span>{" "}
          {shown.join(", ")}
          {hidden > 0 && (
            <>
              {" "}
              <button type="button" className="text-foreground/85 underline-offset-4 hover:underline" onClick={() => setOpen(true)}>
                and {hidden} more
              </button>
            </>
          )}
        </span>
      )}
    </p>
  );
};

export const ProviderButtons = ({ className }: { className?: string }) => (
  <div className={cn("grid gap-2", className)}>
    <Button type="button" className="w-full">
      <GithubMark />
      Continue with GitHub
    </Button>
    <Button type="button" variant="outline" className="w-full">
      <GoogleMark />
      Continue with Google
    </Button>
    <Button type="button" variant="outline" className="w-full">
      <DiscordMark />
      Continue with Discord
    </Button>
  </div>
);

export const SignedInAs = ({ invite, className }: { invite: ProtoInvite; className?: string }) => (
  <div className={cn("flex min-w-0 items-center gap-2.5", className)}>
    <PersonAvatar name={invite.person.name} className="size-6 text-[10px]" />
    <p className="min-w-0 truncate text-[13px]">
      <span className="font-medium">{invite.person.name}</span>
      <span className="font-mono text-xs text-muted-foreground"> @{invite.person.login} · GitHub</span>
    </p>
  </div>
);

export const ExpiryLine = ({ invite, className }: { invite: ProtoInvite; className?: string }) => (
  <p className={cn("font-mono text-[11px] text-muted-foreground", className)}>Works once · expires {invite.expires}</p>
);

export const LATER_LINE = "You may be added to more projects later.";
