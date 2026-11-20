import { z } from "zod";

import { deviceTimezone, nextScheduleSlot, type ScheduleInput, type ScheduledMessage } from "@nexul/client-core/chat";

import { dateInput, instantOf, timeInput } from "@/utils/CalendarUtility";

export const ScheduleFormSchema = z
  .object({
    body: z.string().trim().min(1, "Write the message"),
    date: z.string().min(1, "Pick a day"),
    time: z.string().min(1, "Pick a time"),
    repeats: z.enum(["none", "daily", "weekly", "monthly"]),
  })
  .superRefine((form, ctx) => {
    if (form.date && form.time && Date.parse(instantOf(form.date, form.time)) <= Date.now()) {
      ctx.addIssue({ code: "custom", path: ["time"], message: "Pick a time ahead" });
    }
  });

export type ScheduleFormData = z.infer<typeof ScheduleFormSchema>;

export const newScheduleDefaults = (body: string, now: Date = new Date()): ScheduleFormData => {
  const next = nextScheduleSlot(now).toISOString();
  return { body, date: dateInput(next), time: timeInput(next), repeats: "none" };
};

export const scheduleFormDefaults = (scheduled: ScheduledMessage): ScheduleFormData => ({
  body: scheduled.body,
  date: dateInput(scheduled.send_at),
  time: timeInput(scheduled.send_at),
  repeats: scheduled.repeats,
});

// The form as the routes take it, in the zone the browser runs in.
export const scheduleInput = (form: ScheduleFormData): ScheduleInput => ({
  body: form.body.trim(),
  send_at: instantOf(form.date, form.time),
  repeats: form.repeats,
  timezone: deviceTimezone(),
});

// A change sends a new time only when it moved, so a repeat keeps the start it steps from.
export const scheduleChange = (scheduled: ScheduledMessage, form: ScheduleFormData): Partial<ScheduleInput> => {
  const { send_at, ...rest } = scheduleInput(form);
  return Date.parse(send_at) === Date.parse(scheduled.send_at) ? rest : { ...rest, send_at };
};
