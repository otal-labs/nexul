import { useDeleteBotwebhook, useUpdateBotwebhook } from "@/hooks/BotwebhookHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Botwebhook } from "@/models/Botwebhook";

// Each resolves true once the change landed, so the caller can say what it did; a cancel or a refusal is false.
export const useBotActions = () => {
  const { open: confirm } = useConfirmationDialog();
  const update = useUpdateBotwebhook();
  const remove = useDeleteBotwebhook();

  const settled = (run: Promise<unknown>) =>
    run.then(
      () => true,
      () => false,
    );

  return {
    regenerate: async (bot: Botwebhook) => {
      const ok = await confirm({
        title: `Regenerate ${bot.name}'s URL?`,
        message: "Anything still posting to the old URL gets a 404 from now on, until you paste the new URL into it.",
        confirmLabel: "Regenerate",
      });
      return ok && settled(update.mutateAsync({ bot, change: { regenerate: true } }));
    },
    remove: async (bot: Botwebhook) => {
      const ok = await confirm({
        title: `Delete ${bot.name}?`,
        message: "Anything posting to its URL gets a 404. Its messages stay in the channel, and you can restore it later with a new URL.",
        confirmLabel: "Delete",
      });
      return ok && settled(remove.mutateAsync(bot));
    },
    restore: (bot: Botwebhook) => settled(update.mutateAsync({ bot, change: { deleted: false } })),
  };
};
