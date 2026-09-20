import { UserX } from "lucide-react";

import { NoDataDisplay } from "@/components/NoDataDisplay";
import { LoginProviderMarks } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";

interface AllowlistMemberRowProps {
  login: string;
  onRemove: (login: string) => void;
  removing: boolean;
}

const AllowlistMemberRow = ({ login, onRemove, removing }: AllowlistMemberRowProps) => (
  <li className="flex items-center gap-2 px-3 py-2">
    <LoginProviderMarks login={login} />
    <span className="min-w-0 flex-1 truncate font-mono text-sm">{login}</span>
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={`Remove ${login} from the allowlist`}
      title="Remove from allowlist"
      disabled={removing}
      onClick={() => onRemove(login)}
    >
      <UserX className="size-4" />
    </Button>
  </li>
);

interface AllowlistMembersListProps {
  members: string[];
  onRemove: (login: string) => void;
  removing: boolean;
}

export const AllowlistMembersList = ({ members, onRemove, removing }: AllowlistMembersListProps) => (
  <>
    {members.length === 0 && (
      <NoDataDisplay className="mt-4" message="No one is allowlisted yet besides the instance owner." />
    )}
    {members.length > 0 && (
      <ul className="mt-4 divide-y divide-border overflow-hidden rounded-md border">
        {members.map((login) => (
          <AllowlistMemberRow key={login} login={login} onRemove={onRemove} removing={removing} />
        ))}
      </ul>
    )}
  </>
);
