// Mirrors internal/workspace/model.go's RepoRole; a tests repository is never deployed.
export const RepoRole = {
  App: "app",
  Tests: "tests",
} as const;

export type RepoRole = (typeof RepoRole)[keyof typeof RepoRole];

// Mirrors internal/workspace/model.go's TestsLocation; "" means the question has not been answered.
export const TestsLocation = {
  Unset: "",
  Same: "same",
  Separate: "separate",
} as const;

export type TestsLocation = (typeof TestsLocation)[keyof typeof TestsLocation];
