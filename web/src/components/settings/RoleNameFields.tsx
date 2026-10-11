import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Role } from "@/models/Role";
import { cn } from "@/lib/utils";

interface RoleNameFieldsProps {
  creating: boolean;
  name: string;
  copiedFrom: Role | undefined;
  onName: (name: string) => void;
  onStartFrom: (role: Role | undefined) => void;
}

const NOTHING = "nothing";

// The name, and on a new role a "Start from" that copies another role's permissions into the draft.
export const RoleNameFields = ({ creating, name, copiedFrom, onName, onStartFrom }: RoleNameFieldsProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: roles } = useFetchWorkspaceRoles(workspaceId);
  const copyable = (roles ?? []).filter((role) => !role.is_owner_role);

  return (
    <div className="@container space-y-1.5">
      <div className={cn("grid gap-3", creating && "@lg:grid-cols-[minmax(0,1fr)_12rem]")}>
        <label className="grid gap-1.5">
          <span className="text-sm font-medium">Name</span>
          <Input value={name} placeholder="Support, Contractor…" onChange={(event) => onName(event.target.value)} />
        </label>
        {creating && (
          <div className="grid gap-1.5">
            <span id="role-start-from" className="text-sm font-medium">
              Start from
            </span>
            <Select
              value={copiedFrom?.id ?? NOTHING}
              onValueChange={(id) => onStartFrom(copyable.find((role) => role.id === id))}
            >
              <SelectTrigger aria-labelledby="role-start-from" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NOTHING}>Nothing allowed</SelectItem>
                {copyable.map((role) => (
                  <SelectItem key={role.id} value={role.id}>
                    A copy of {role.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>
      {copiedFrom && (
        <p className="text-xs text-muted-foreground">
          Copied from {copiedFrom.name}. Change anything below; {copiedFrom.name} stays as it is.
        </p>
      )}
    </div>
  );
};
