import { createContext, use, type ActionDispatch } from "react";

import type { MemberDraft, MemberDraftAction } from "@/models/MemberDraft";
import type { TeamPerson } from "@/models/Team";

interface MemberDraftContextValue {
  person: TeamPerson;
  draft: MemberDraft;
  dispatch: ActionDispatch<[MemberDraftAction]>;
}

// The Team dialog's person and held changes, provided by the dialog so its panels read and edit them without threading.
export const MemberDraftContext = createContext<MemberDraftContextValue | null>(null);

export const useMemberDraft = (): MemberDraftContextValue => {
  const value = use(MemberDraftContext);
  if (!value) throw new Error("useMemberDraft needs the Team dialog around it");
  return value;
};
