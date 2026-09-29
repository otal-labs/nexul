import { deployTitle, type Deploy } from "@/models/Stack";

const deploy = (image: string): Deploy => ({
  id: "0198f2ab-1111-2222-3333-444455556666",
  stack_id: "st-1",
  image,
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
    ["", "Build 0198f2a"],
  ])("names %j as %j", (image, title) => {
    expect(deployTitle(deploy(image))).toBe(title);
  });
});
