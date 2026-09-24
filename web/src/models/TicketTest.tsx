import { z } from "zod";

// preview is the branch's own deployment; shared is an environment other work also lands on.
export const TestTargetKind = {
  Preview: "preview",
  Shared: "shared",
} as const;

export type TestTargetKind = (typeof TestTargetKind)[keyof typeof TestTargetKind];

// An empty url means no deployment is safe to test on; the server never returns production.
export interface TestTarget {
  url: string;
  kind?: TestTargetKind;
  branch?: string;
}

const ScreenshotShape = z.object({ id: z.string(), name: z.string() });

export type Screenshot = z.infer<typeof ScreenshotShape>;

// The bug template's sections; screenshots are ticket attachments uploaded while the dialog is open.
export const TestFailFormSchema = z.object({
  steps: z.string(),
  expected: z.string(),
  actual: z.string().trim().min(1, "Say what went wrong"),
  screenshots: z.array(ScreenshotShape),
});

export type TestFailFormData = z.infer<typeof TestFailFormSchema>;

export const emptyTestFailForm = (): TestFailFormData => ({ steps: "", expected: "", actual: "", screenshots: [] });
