import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { UserPlus } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AllowlistMembersList } from "@/components/settings/AllowlistMembersList";
import { AllowlistSuggestionsList } from "@/components/settings/AllowlistSuggestionsList";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAddMember, useFetchMembers, useLookupMembers, useRemoveMember } from "@/hooks/AuthHooks";
import { AddMemberFormSchema, type AddMemberFormData, type LoginMatch } from "@/models/User";

// Instance-wide sign-in allowlist (ADR 0040) — deliberately separate from workspace membership/roles.
export const AllowlistSection = () => {
  const { data: members, isPending, error } = useFetchMembers();
  const addMember = useAddMember();
  const removeMember = useRemoveMember();
  const form = useForm<AddMemberFormData>({
    defaultValues: { login: "" },
    resolver: zodResolver(AddMemberFormSchema),
  });

  const loginValue = useWatch({ control: form.control, name: "login" }) || "";
  // The one value the suggestion list was closed for (Escape, or a picked login); editing past it reopens the list.
  const [hiddenFor, setHiddenFor] = useState<string | null>(null);

  const { data: lookup } = useLookupMembers(loginValue);
  const matches = lookup?.matches ?? [];
  // keepPreviousData would resurface stale matches after reset/email; gate on the live value too.
  const lookupShaped = loginValue.trim().length >= 2 && !loginValue.includes("@");
  const showSuggestions = lookupShaped && matches.length > 0 && loginValue !== hiddenFor;

  const onSubmit = async (input: AddMemberFormData) => {
    try {
      await addMember.mutateAsync(input.login);
      form.reset({ login: "" });
    } catch {
      // Error is surfaced by the hook's toast; the input keeps its value.
    }
  };

  const pickSuggestion = (match: LoginMatch) => {
    form.setValue("login", match.login, { shouldValidate: true, shouldDirty: true });
    setHiddenFor(match.login);
  };

  return (
    <SettingsCard
      id="instance-access"
      title="Instance access"
      description="Only allowlisted GitHub usernames or Google/Discord emails can sign in to this instance at all. This is separate from workspace membership — being allowlisted doesn't put anyone in a workspace by itself. Start typing a GitHub username to get suggestions; emails have no lookup."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {members && (
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-1.5">
          <div className="flex flex-col gap-2 rounded-md border bg-card p-1.5 shadow-card sm:flex-row sm:items-center">
            <Controller
              control={form.control}
              name="login"
              render={({ field }) => (
                <Input
                  aria-label="GitHub username or email"
                  placeholder="GitHub username or Google/Discord email to allow"
                  aria-invalid={form.formState.errors.login != null}
                  className="border-0 bg-transparent shadow-none sm:flex-1"
                  {...field}
                  onKeyDown={(event) => {
                    if (event.key === "Escape") setHiddenFor(loginValue);
                  }}
                />
              )}
            />
            <Button type="submit" disabled={form.formState.isSubmitting}>
              <UserPlus className="size-4" />
              Allow
            </Button>
          </div>
          <AllowlistSuggestionsList show={showSuggestions} matches={matches} onPick={pickSuggestion} />
          {form.formState.errors.login && (
            <p role="alert" className="text-sm text-destructive">
              {form.formState.errors.login.message}
            </p>
          )}
        </form>
      )}
      {members && (
        <AllowlistMembersList
          members={members.members}
          onRemove={(login) => removeMember.mutate(login)}
          removing={removeMember.isPending}
        />
      )}
    </SettingsCard>
  );
};
