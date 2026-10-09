import { describe, expect, it } from "vitest";

import { interviewSourceFollower } from "@/hooks/InterviewSourceHooks";
import type { InterviewSource } from "@/models/InterviewSource";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const source = (ref: string) => ({ id: `src-${ref}`, workspace_id: "ws-1", project_id: "p-1", kind: "doc", ref, label: ref, stance: "follow" }) as InterviewSource;

describe("the interview source follower", () => {
  it("refetches a project's sources only when a doc or memory it names changes", async () => {
    const client = seeded([[["getInterviewSources", "p-1"], [source("d-1")]]]);
    await followFrame(interviewSourceFollower, "doc.updated", { doc: { id: "d-9", project_id: "p-1" } }, client);
    expect(isStale(client, ["getInterviewSources", "p-1"])).toBe(false);

    await followFrame(interviewSourceFollower, "doc.updated", { doc: { id: "d-1", project_id: "p-1" } }, client);
    expect(isStale(client, ["getInterviewSources", "p-1"])).toBe(true);
  });

  it("refetches the sources and drafts of the project a frame names, and no other", async () => {
    const client = seeded([
      [["getInterviewSources", "p-1"], []],
      [["getInterviewSources", "p-2"], []],
      [["getInterviewDrafts", "p-1"], []],
    ]);
    await followFrame(interviewSourceFollower, "interview_source.added", { workspace_id: "ws-1", project_id: "p-2", source_id: "src-1" }, client);
    await followFrame(interviewSourceFollower, "interview_draft.saved", { workspace_id: "ws-1", project_id: "p-1", draft_id: "dr-1" }, client);
    const keys = [["getInterviewSources", "p-1"], ["getInterviewSources", "p-2"], ["getInterviewDrafts", "p-1"]];
    expect(keys.map((key) => isStale(client, key))).toEqual([false, true, true]);
  });
});
