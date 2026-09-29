import { useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { PermissionGrid } from "@/components/access/PermissionGrid";
import { PermissionLevels } from "@/components/access/PermissionLevels";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useSetTeamMemberOverrides } from "@/hooks/TeamHooks";
import type { TeamMembership } from "@/models/Team";

interface TeamOverridesFormProps {
  target: { workspaceId: string; userId: string };
  membership: TeamMembership;
  onDone: () => void;
}

interface OverridesFormData {
  allow: string[];
  deny: string[];
}

// The same allow levels and deny grid an invitation uses, applied to a member who already joined.
export const TeamOverridesForm = ({ target, membership, onDone }: TeamOverridesFormProps) => {
  const { data: catalog, isPending, error } = useFetchPermissionCatalog();
  const save = useSetTeamMemberOverrides();
  const form = useForm<OverridesFormData>({ defaultValues: { allow: membership.allow, deny: membership.deny } });
  const [allow, deny] = useWatch({ control: form.control, name: ["allow", "deny"] });
  const overlap = allow.some((value) => deny.includes(value));

  return (
    <form className="space-y-3" onSubmit={form.handleSubmit((values) => save.mutate({ ...target, ...values }, { onSuccess: onDone }))}>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {catalog && (
        <div className="space-y-1">
          <p className="text-xs font-medium">Allow</p>
          <PermissionLevels entries={catalog} value={allow} onChange={(value) => form.setValue("allow", value, { shouldDirty: true })} />
        </div>
      )}
      {catalog && (
        <div className="space-y-1">
          <p className="text-xs font-medium">Deny</p>
          <PermissionGrid entries={catalog} value={deny} onChange={(value) => form.setValue("deny", value, { shouldDirty: true })} />
        </div>
      )}
      {overlap && <p role="alert" className="text-sm text-destructive">A permission can&apos;t be allowed and denied at once.</p>}
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={!form.formState.isDirty || overlap || save.isPending}>
          Save overrides
        </Button>
        <Button type="button" variant="ghost" size="sm" onClick={onDone}>
          Cancel
        </Button>
      </div>
    </form>
  );
};
