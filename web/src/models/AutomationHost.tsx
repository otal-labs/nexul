import { z } from "zod";

import { hasPermission } from "@nexul/client-core/permissions";

import type { Machine } from "@/models/Runner";

// The bundled host every unplaced automation runs on.
export const InstanceHostName = "instance";

export const AutomationHostSchema = z.object({
  id: z.string().min(1),
  name: z.string(),
  machine: z.string(),
  os: z.string(),
  arch: z.string(),
  version: z.string(),
  // true when the host polled for its automations within the last minute.
  connected: z.boolean(),
  last_seen: z.string(),
});

export type AutomationHost = z.infer<typeof AutomationHostSchema>;

// The machine card a host answers to, or undefined for a host on a machine Nexul doesn't know, which only the instance Owner sees.
export const machineOfHost = (host: AutomationHost, machines: readonly Machine[] | undefined): Machine | undefined =>
  machines?.find((machine) => machine.name === host.machine);

// Whether the viewer may place an automation on host: Deploy to on its machine, or a machine Nexul doesn't know, which only the Owner is shown.
export const canPlaceOn = (host: AutomationHost, machines: readonly Machine[]): boolean => {
  const machine = machineOfHost(host, machines);
  return !machine || hasPermission(machine.actions, "machines:deploy");
};

// Names become service names on the machine, so they share the installer's pattern.
export const AutomationHostEnrollmentFormSchema = z.object({
  name: z
    .string()
    .trim()
    .regex(/^[a-z0-9][a-z0-9-]{0,31}$/, "Use 1 to 32 lowercase letters, digits or dashes"),
  machine: z.string().trim(),
});

export type AutomationHostEnrollmentFormData = z.infer<typeof AutomationHostEnrollmentFormSchema>;

export const AutomationHostEnrollmentSchema = z.object({
  code: z.string(),
  expires_at: z.string(),
  commands: z.object({ unix: z.string(), windows: z.string() }),
});

export type AutomationHostEnrollment = z.infer<typeof AutomationHostEnrollmentSchema>;
