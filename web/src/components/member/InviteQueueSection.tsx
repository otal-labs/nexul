import { zodResolver } from "@hookform/resolvers/zod";
import { UserPlus } from "lucide-react";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";

import { PendingInviteRow } from "@/components/member/PendingInviteRow";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { InviteMemberFormSchema, type InviteMemberFormData, type PendingInvite } from "@/models/Member";
import type { Role } from "@/models/Role";

interface InviteQueueSectionProps {
  assignableRoles: Role[];
  existingLogins: string[];
  onInvite: (entry: PendingInvite) => Promise<unknown>;
}

// Usernames pile up here before one "Continue" submits them all, so one bad login skips retyping the rest.
export const InviteQueueSection = ({ assignableRoles, existingLogins, onInvite }: InviteQueueSectionProps) => {
  const [queue, setQueue] = useState<PendingInvite[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const form = useForm<InviteMemberFormData>({
    defaultValues: { login: "" },
    resolver: zodResolver(InviteMemberFormSchema),
  });

  const alreadyQueuedOrInvited = new Set([...queue.map((q) => q.login), ...existingLogins]);

  const onQueue = (input: InviteMemberFormData) => {
    const login = input.login.trim().toLowerCase();
    if (alreadyQueuedOrInvited.has(login)) {
      form.setError("login", { message: "Already queued or a member of this workspace" });
      return;
    }
    setQueue((q) => [...q, { login, roleId: assignableRoles[0]?.id ?? "" }]);
    form.reset({ login: "" });
  };

  const onSubmitQueue = async () => {
    setSubmitting(true);
    const results = await Promise.allSettled(queue.map((entry) => onInvite(entry)));
    setQueue((q) => q.filter((_, i) => results[i]?.status === "rejected"));
    setSubmitting(false);
  };

  return (
    <>
      <form onSubmit={form.handleSubmit(onQueue)} className="mt-6 space-y-1.5">
        <div className="flex flex-col gap-2 rounded-md border bg-card p-1.5 shadow-card sm:flex-row sm:items-center">
          <Controller
            control={form.control}
            name="login"
            render={({ field }) => (
              <Input
                aria-label="GitHub username"
                placeholder="colleague@company.com"
                aria-invalid={form.formState.errors.login != null}
                className="border-0 bg-transparent shadow-none sm:flex-1"
                {...field}
              />
            )}
          />
          <Button type="submit">
            <UserPlus className="size-4" />
            Add
          </Button>
        </div>
        {form.formState.errors.login && (
          <p role="alert" className="text-sm text-destructive">
            {form.formState.errors.login.message}
          </p>
        )}
      </form>

      {queue.length > 0 && (
        <div className="mt-4 space-y-2">
          <p className="text-sm font-medium text-muted-foreground">Will invite ({queue.length})</p>
          <ul className="space-y-1.5">
            {queue.map((entry) => (
              <PendingInviteRow
                key={entry.login}
                login={entry.login}
                roleId={entry.roleId}
                roles={assignableRoles}
                onRoleChange={(login, roleId) =>
                  setQueue((q) => q.map((e) => (e.login === login ? { ...e, roleId } : e)))
                }
                onRemove={(login) => setQueue((q) => q.filter((e) => e.login !== login))}
              />
            ))}
          </ul>
          <Button type="button" className="w-full" disabled={submitting} onClick={() => void onSubmitQueue()}>
            {submitting ? "Inviting…" : "Continue"}
          </Button>
        </div>
      )}
    </>
  );
};
