// Mirrors memories.Question: one ## heading of the Interview template, parsed on the server.
export interface InterviewQuestion {
  text: string;
  hint: string;
  multi_select: boolean;
  options: { label: string; description: string }[];
}

export interface InterviewTemplate {
  workspace_id: string;
  /** Markdown list of the questions a project's interview asks, one ## heading each. */
  body: string;
  questions: InterviewQuestion[];
  /** The instance's Interview template, which an unedited workspace follows and a reset returns to. */
  default_body: string;
  /** False while the workspace follows the instance template. */
  edited: boolean;
  updated_by: string;
  updated_at: string;
}
