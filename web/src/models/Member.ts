// Mirrors internal/tenancy/model.go's MemberView/Invite/MembersList wire shapes (Membership invites' Members page).
export interface MemberView {
  user_id: string;
  login: string;
  role_id: string;
}

export interface MembersList {
  members: MemberView[];
}
