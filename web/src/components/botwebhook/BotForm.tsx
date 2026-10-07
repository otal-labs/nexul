import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronLeft } from "lucide-react";
import { useForm, useWatch } from "react-hook-form";

import { errorMessage } from "@/api/client";
import { BotAvatarControl } from "@/components/botwebhook/BotAvatarControl";
import { BotDetailFooter } from "@/components/botwebhook/BotDetailFooter";
import { BotUrlBlock } from "@/components/botwebhook/BotUrlBlock";
import { FormInput } from "@/components/FormInput";
import { useCreateBotwebhook, useFetchBotwebhooks, useUpdateBotwebhook } from "@/hooks/BotwebhookHooks";
import { botFormSchema, type BotFormData, type Botwebhook } from "@/models/Botwebhook";

// Which bot the section shows in place of its list, "new" for the empty create view, and the line after a URL change.
export interface BotView {
  id: string;
  note?: string;
}

interface BotFormProps {
  conversationId: string;
  /** Undefined is the create view. */
  bot?: Botwebhook | undefined;
  note?: string | undefined;
  onNavigate: (view: BotView | null) => void;
}

export const BotForm = ({ conversationId, bot, note, onNavigate }: BotFormProps) => {
  const { data: bots } = useFetchBotwebhooks(conversationId);
  const create = useCreateBotwebhook(conversationId);
  const update = useUpdateBotwebhook();
  const taken = (bots ?? []).filter((other) => other.id !== bot?.id).map((other) => other.name);
  const form = useForm<BotFormData>({
    defaultValues: { name: bot?.name ?? "", avatar: bot?.avatar ?? "" },
    resolver: zodResolver(botFormSchema(taken)),
    mode: "onChange",
  });
  const name = useWatch({ control: form.control, name: "name" });
  const draftAvatar = useWatch({ control: form.control, name: "avatar" });
  const renamed = !!bot && name.trim() !== bot.name;
  const serverError = form.formState.errors.root?.message;

  const onSubmit = async (data: BotFormData) => {
    try {
      if (!bot) {
        const created = await create.mutateAsync({ name: data.name, avatar: data.avatar });
        onNavigate({ id: created.id });
        return;
      }
      await update.mutateAsync({ bot, change: { name: data.name } });
    } catch (error) {
      form.setError("root", { message: errorMessage(error) });
    }
  };

  const onAvatar = (avatar: string) => {
    if (!bot) {
      form.setValue("avatar", avatar);
      return;
    }
    update.mutate({ bot, change: { avatar } });
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
      <button
        type="button"
        onClick={() => onNavigate(null)}
        className="-ml-1 flex items-center gap-1 rounded-md px-1 text-sm text-muted-foreground hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
      >
        <ChevronLeft className="size-4" aria-hidden />
        Bots
      </button>
      <BotAvatarControl avatar={bot ? (bot.avatar ?? "") : draftAvatar} onChange={onAvatar} />
      <FormInput control={form.control} name="name" id="bot-name" label="Name" placeholder="Name, such as CI" autoComplete="off" autoFocus={!bot} />
      {bot && <BotUrlBlock bot={bot} note={note} />}
      {serverError && (
        <p role="alert" className="text-xs text-destructive">
          {serverError}
        </p>
      )}
      <BotDetailFooter
        bot={bot}
        submitLabel={bot ? "Save name" : "Create"}
        showSubmit={!bot || renamed}
        submitting={form.formState.isSubmitting}
        onNavigate={onNavigate}
      />
    </form>
  );
};
