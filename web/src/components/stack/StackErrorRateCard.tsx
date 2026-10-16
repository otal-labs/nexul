import { zodResolver } from "@hookform/resolvers/zod";
import { ActivityIcon } from "lucide-react";
import { useForm } from "react-hook-form";

import { ActionSwitch } from "@/components/action/ActionSwitch";
import { FormInput } from "@/components/FormInput";
import { GatedSaveBar } from "@/components/settings/GatedSaveBar";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchStackRequests, useUpdateStack } from "@/hooks/StackHooks";
import { useActionState } from "@/hooks/useActionState";
import { useFlash } from "@/hooks/useFlash";
import { useStackActions } from "@/hooks/useStackActions";
import { ErrorRateFormSchema, type ErrorRateFormData, type Stack } from "@/models/Stack";
import { errorRateOf, recoveryText, ruleText } from "@/utils/ErrorRateUtility";

const FORM_ID = "stack-error-rate-form";
const SWITCH = "Alert when 5xx errors stay high";

interface StackErrorRateCardProps {
  stack: Stack;
}

// Saving sends the whole stack, so a reader whose env values were withheld must not save it and blank them.
const reasonOff = (stack: Stack, unavailable: string | undefined): string | undefined => {
  if (stack.env_masked) return "Your role can't see this stack's variables";
  return unavailable || undefined;
};

// One rule per stack, off until someone turns it on: what trips it in the switch row, its numbers below, how it ends in the footer.
export const StackErrorRateCard = ({ stack }: StackErrorRateCardProps) => {
  const rule = errorRateOf(stack);
  const answer = useStackActions(stack);
  const update = useUpdateStack();
  const { data: requests } = useFetchStackRequests(stack.id);
  const [saved, flash] = useFlash();
  const writable = useActionState({ permission: "stacks:write", projectId: stack.project_id || undefined, answer: answer("stacks:write") }).state === "live";
  const form = useForm<ErrorRateFormData>({
    disabled: !writable,
    values: { percent: rule.percent, minutes: rule.minutes, min_requests: rule.min_requests },
    resolver: zodResolver(ErrorRateFormSchema),
  });
  const off = reasonOff(stack, requests?.error_rate_unavailable);
  // A rule that is already on can always be switched off, whatever has gone wrong around it.
  const blocked = !!off && !rule.enabled;

  const onSubmit = async (data: ErrorRateFormData) => {
    try {
      await update.mutateAsync({ ...stack, error_rate: { ...rule, ...data } });
      form.reset(data);
      flash();
    } catch {
      // Error is surfaced by the hook's toast; the numbers stay in the form to retry.
    }
  };

  return (
    <SettingsCard
      id="error-rate"
      title="Error rate"
      icon={ActivityIcon}
      footer={
        <GatedSaveBar
          permission="stacks:write"
          projectId={stack.project_id || undefined}
          answer={answer("stacks:write")}
          verb="change this stack's alerts"
          form={FORM_ID}
          dirty={form.formState.isDirty}
          saving={form.formState.isSubmitting}
          saved={saved}
          onDiscard={() => form.reset()}
          hint={recoveryText(rule)}
        />
      }
    >
      <div className="space-y-5">
        <div className="flex items-start gap-3">
          <div className="min-w-0 flex-1 space-y-0.5">
            <p className="text-sm font-medium">{SWITCH}</p>
            <p className="text-sm text-muted-foreground">{blocked ? off : `Trips on ${ruleText(rule)}. Everyone watching the stack is told when it trips and when it ends.`}</p>
          </div>
          <ActionSwitch
            permission="stacks:write"
            projectId={stack.project_id || undefined}
            answer={answer("stacks:write")}
            verb="change this stack's alerts"
            checked={rule.enabled}
            disabled={blocked || update.isPending || !!stack.env_masked}
            onCheckedChange={(enabled) => update.mutate({ ...stack, error_rate: { ...rule, enabled } })}
            aria-label={SWITCH}
            {...(blocked && { title: off })}
            className="mt-0.5"
          />
        </div>
        <form id={FORM_ID} onSubmit={form.handleSubmit(onSubmit)} className="grid grid-cols-1 gap-3 @md:grid-cols-3">
          <FormInput control={form.control} name="percent" label="5xx above (%)" type="number" step="any" inputMode="decimal" />
          <FormInput control={form.control} name="minutes" label="For (minutes)" type="number" inputMode="numeric" />
          <FormInput control={form.control} name="min_requests" label="With at least (requests)" type="number" inputMode="numeric" />
        </form>
      </div>
    </SettingsCard>
  );
};
