import { deployTitle, type Deploy } from "@/models/Stack";

const deploy = (image: string, commit?: string): Deploy => ({
  id: "0198f2ab-1111-2222-3333-444455556666",
  stack_id: "st-1",
  image,
  ...(commit && { commit }),
  status: "healthy",
  created_at: "2026-09-01T00:00:00Z",
  updated_at: "2026-09-01T00:00:00Z",
});

describe("deployTitle", () => {
  test.each([
    ["ghcr.io/org/app:abc123", "abc123"],
    ["localhost:5000/app:v2", "v2"],
    ["ghcr.io/org/app@sha256:9f86d081884c7d65", "9f86d08"],
    ["nginx", "nginx"],
    ["", "Repo build"],
  ])("names %j as %j", (image, title) => {
    expect(deployTitle(deploy(image))).toBe(title);
  });
});

test("names a build with no image by its commit, never by the deploy's id", () => {
  expect(deployTitle(deploy("", "9c1e5a77d2b04f3e8a6c1d0b2e4f6a8c0d2e4f6a"))).toBe("Build 9c1e5a7");
});
