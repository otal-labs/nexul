import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DiscordMark, GithubMark, GoogleMark } from "@/components/ProviderMarks";
import { IdentityRow } from "@/components/you/IdentityRow";
import { useFetchMe } from "@/hooks/AuthHooks";
import { effectiveAvatar } from "@/models/User";

export const ProfileSection = () => {
  const { data: me } = useFetchMe();
  const user = me?.user;

  return (
    <>
      <SettingsCard
        id="profile"
        title="Profile"
        description="The name and picture everyone in your workspaces sees."
        footer={
          <>
            <p className="text-sm text-muted-foreground">Leave a field empty to use your sign-in account's.</p>
            <Button size="sm">Save</Button>
          </>
        }
      >
        <div className="flex items-start gap-5">
          <span className="size-16 shrink-0 overflow-hidden rounded-lg bg-accent">
            {user && effectiveAvatar(user) !== "" && (
              <img src={effectiveAvatar(user)} alt="" referrerPolicy="no-referrer" className="size-full object-cover" />
            )}
          </span>
          <div className="grid min-w-0 flex-1 gap-4">
            <div className="grid gap-1.5">
              <label htmlFor="display-name" className="text-sm font-medium">Display name</label>
              <Input id="display-name" placeholder={user?.name || user?.login} defaultValue={user?.display_name} />
            </div>
            <div className="grid gap-1.5">
              <label htmlFor="avatar-url" className="text-sm font-medium">Picture URL</label>
              <Input id="avatar-url" placeholder="https://…" defaultValue={user?.avatar_override_url} />
            </div>
          </div>
        </div>
      </SettingsCard>
      <SettingsCard
        id="sign-in-accounts"
        title="Sign-in accounts"
        description="Any linked account signs you in to this same profile."
      >
        <ul className="divide-y divide-border overflow-hidden rounded-md border border-border">
          <IdentityRow mark={GithubMark} provider="GitHub" account={user ? `@${user.login}` : undefined} />
          <IdentityRow mark={GoogleMark} provider="Google" account="onik@example.com" />
          <IdentityRow mark={DiscordMark} provider="Discord" />
        </ul>
      </SettingsCard>
    </>
  );
};
