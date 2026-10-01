export interface InterviewTemplate {
  workspace_id: string;
  /** Markdown a new project's interview memory is copied from once. */
  body: string;
  /** The instance's Interview template, which an unedited workspace follows and a reset returns to. */
  default_body: string;
  /** False while the workspace follows the instance template. */
  edited: boolean;
  updated_by: string;
  updated_at: string;
}
