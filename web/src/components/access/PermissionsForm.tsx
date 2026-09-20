import { useEffect, useMemo, useState } from "react";
import { z } from "zod";

import { EmptyState } from "@/components/EmptyState";
import { PermissionCheckRow } from "@/components/access/PermissionCheckRow";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { useFetchPermissionCatalog, useFetchPermissionUsers, useSetPermissions } from "@/hooks/PermissionHooks";

export const PermissionsFormSchema = z.object({});

export type PermissionsFormData = z.infer<typeof PermissionsFormSchema>;

interface PermissionsFormProps {
  resourceType: "doc" | "play";
  resourceIds: string[];
}

// Generalised over resource type (ticket 21): a "doc" submission keeps the exact pre-existing wire shape
// (doc_ids, no resource_type field) so the docs sharing dialog is unaffected; a "play" submission denies
// or un-denies plays:run, the one meaningful action on a play.
export const PermissionsForm = ({ resourceType, resourceIds }: PermissionsFormProps) => {
  const { onSubmit, setLoading } = useFormDialogContext<PermissionsFormData>();
  const { data: users } = useFetchPermissionUsers();
  const { data: catalog } = useFetchPermissionCatalog();
  const setPermissions = useSetPermissions();
  const [selectedUsers, setSelectedUsers] = useState<string[]>([]);
  const [selectedActions, setSelectedActions] = useState<string[]>([]);
  const [grant, setGrant] = useState(resourceType !== "play");

  // Doc permissions cover the docs domain plus the cross-cutting permissions:write grant; a play offers
  // only plays:run, since a deny of that is the only meaningful overwrite on a play (ticket 21).
  const permissionOptions = useMemo(() => {
    if (resourceType === "play") {
      return (catalog ?? []).filter((entry) => entry.value === "plays:run");
    }
    return (catalog ?? []).filter((entry) => entry.domain === "docs" || entry.value === "permissions:write");
  }, [catalog, resourceType]);

  const canSubmit = resourceIds.length > 0 && selectedUsers.length > 0 && selectedActions.length > 0;

  // The shell's OK button is Apply; disabled until a user and an action are picked (bulk state, not RHF).
  useEffect(() => {
    setLoading(!canSubmit);
  }, [canSubmit, setLoading]);

  onSubmit(async () => {
    if (resourceType === "play") {
      await setPermissions.mutateAsync({
        resource_type: "play",
        resource_ids: resourceIds,
        user_ids: selectedUsers,
        actions: selectedActions,
        grant,
      });
      return {};
    }
    await setPermissions.mutateAsync({
      doc_ids: resourceIds,
      user_ids: selectedUsers,
      actions: selectedActions,
      grant,
    });
    return {};
  });

  const toggleUser = (id: string) => {
    setSelectedUsers((prev) => (prev.includes(id) ? prev.filter((u) => u !== id) : [...prev, id]));
  };

  const toggleAction = (action: string) => {
    setSelectedActions((prev) =>
      prev.includes(action) ? prev.filter((a) => a !== action) : [...prev, action],
    );
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {resourceIds.length} {resourceType === "play" ? "play" : "document"}
        {resourceIds.length === 1 ? "" : "s"} × {selectedUsers.length} user
        {selectedUsers.length === 1 ? "" : "s"} × {selectedActions.length} action
        {selectedActions.length === 1 ? "" : "s"}
      </p>

      <section className="space-y-2">
        <h3 className="text-sm font-semibold tracking-tight">Users</h3>
        <div className="max-h-40 space-y-1 overflow-y-auto rounded-lg border p-2">
          {users?.map((user) => (
            <PermissionCheckRow
              key={user.id}
              checked={selectedUsers.includes(user.id)}
              onCheckedChange={() => toggleUser(user.id)}
            >
              <span className="flex-1">{user.login}</span>
              {user.can_create_workspace && (
                <span className="text-xs text-muted-foreground">owner</span>
              )}
            </PermissionCheckRow>
          ))}
          {users?.length === 0 && <EmptyState title="No users yet" className="border-0 p-3" />}
        </div>
      </section>

      <section className="space-y-2">
        <h3 className="text-sm font-semibold tracking-tight">Actions</h3>
        <div className="grid grid-cols-1 gap-1 rounded-lg border p-2">
          {permissionOptions.map((entry) => (
            <PermissionCheckRow
              key={entry.value}
              checked={selectedActions.includes(entry.value)}
              onCheckedChange={() => toggleAction(entry.value)}
            >
              <span>{entry.label}</span>
            </PermissionCheckRow>
          ))}
        </div>
      </section>

      <section className="space-y-2">
        <h3 className="text-sm font-semibold tracking-tight">Mode</h3>
        <PermissionCheckRow
          checked={grant}
          onCheckedChange={(checked) => setGrant(checked === true)}
          className="rounded-lg border p-2"
        >
          {resourceType === "play" ? "Allow (checked) / Deny (unchecked)" : "Grant (checked) / Revoke (unchecked)"}
        </PermissionCheckRow>
      </section>
    </div>
  );
};
