import { zodResolver } from "@hookform/resolvers/zod";
import { ShieldCheck } from "lucide-react";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useAnswerAgentSecret } from "@/hooks/AgentSecretHooks";
import { SaveSecretFormSchema, type HarnessSecretRequest, type SaveSecretFormData } from "@/models/Secret";

interface ChatSecretCardProps {
  conversationId: string;
  request: HarnessSecretRequest;
  // answered means the thread already moved past this request; the card then only shows what was asked.
  answered: boolean;
}

// The Agent asking for a secret: saving hands the value to its harness, which keeps it out of the chat.
export const ChatSecretCard = ({ conversationId, request, answered }: ChatSecretCardProps) => {
  const answer = useAnswerAgentSecret(conversationId);
  const form = useForm<SaveSecretFormData>({ defaultValues: { secret: "" }, resolver: zodResolver(SaveSecretFormSchema) });
  const open = !answered && !answer.isSuccess;

  const save = async ({ secret }: SaveSecretFormData) => {
    await answer.mutateAsync({ requestId: request.id, secret });
    form.reset();
  };

  return (
    <div role="group" aria-label="Secret requested by the Agent" className="max-w-[85%] rounded-lg border border-border bg-card p-3">
      <h4 className="text-sm font-medium text-balance">{request.label}</h4>
      {request.reason && <p className="mt-1 text-xs text-pretty text-muted-foreground">{request.reason}</p>}
      {open && (
        <form onSubmit={form.handleSubmit(save)} className="mt-3 grid grid-cols-[1fr_auto] items-start gap-2">
          <FormInput
            control={form.control}
            name="secret"
            id={`secret-${request.id}`}
            label={request.label}
            hideLabel
            type="password"
            placeholder={request.placeholder}
            className="w-full font-mono"
            autoComplete="off"
            data-1p-ignore
          />
          <Button type="submit" size="sm" loading={answer.isPending}>
            Save securely
          </Button>
        </form>
      )}
      {open && (
        <div className="mt-2 flex items-center justify-between gap-2">
          <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <ShieldCheck className="size-3.5 shrink-0" aria-hidden />
            Sent to the agent's computer, never kept in Nexul
          </p>
          <Button type="button" variant="ghost" size="sm" disabled={answer.isPending} onClick={() => answer.mutate({ requestId: request.id, secret: "" })}>
            Decline
          </Button>
        </div>
      )}
      {!open && <p className="mt-2 text-xs text-muted-foreground">Answered</p>}
    </div>
  );
};
