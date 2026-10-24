import { defineAutomation } from "@otal-labs/nexul-sdk/automation";

interface IntakeResult {
  outcome: string;
  ticket_id?: string;
  reason?: string;
}

const automation = defineAutomation({
  name: "Crash intake",
  description:
    "Files each crash report a bot posts in a channel as one bug ticket, notes repeats on it, and files a high-priority bug when a fixed crash comes back.",
  config: {
    channel: { type: "channel", label: "Crash report channel", required: true },
    projectId: { type: "project", label: "Project", required: true },
    bugType: { type: "string", label: "Bug ticket type", default: "bug" },
    label: { type: "string", label: "High-priority label", default: "high-priority" },
  },
});

// The rules live in the server's intake, which can post notes for a default automation that has no creator.
automation.on("chat.bot_message.posted", async (payload, ctx) => {
  const { channel, projectId, bugType, label } = ctx.config;
  if (!channel || !projectId) {
    ctx.log("skipped: no crash report channel or project configured", { channel, projectId });
    return false;
  }
  if (payload.conversation_id !== channel) return true;
  const result = await ctx.api.request<IntakeResult>("POST", "/api/crash-reports", {
    project_id: projectId,
    bug_type: bugType,
    label,
    content: payload.content,
    embeds: payload.embeds ?? [],
  });
  ctx.log(`crash report ${result.outcome}`, { ticketId: result.ticket_id, reason: result.reason });
  return true;
});

export default automation;
