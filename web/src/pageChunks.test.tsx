import { describe, expect, it, vi } from "vitest";

// Records which watched modules the app's static import graph evaluates; anything evaluated here ships in the entry chunk.
const { evaluated, watch } = vi.hoisted(() => {
  const evaluated = new Set<string>();
  const watch = (name: string) => async (importOriginal: () => Promise<unknown>) => {
    evaluated.add(name);
    return importOriginal();
  };
  return { evaluated, watch };
});
vi.mock("@/pages/BoardPage", watch("BoardPage"));
vi.mock("@/pages/ChatPage", watch("ChatPage"));
vi.mock("@/pages/DocsPage", watch("DocsPage"));
vi.mock("@/pages/LoginPage", watch("LoginPage"));
vi.mock("@tiptap/core", watch("@tiptap/core"));
vi.mock("@xyflow/react", watch("@xyflow/react"));
vi.mock("livekit-client", watch("livekit-client"));
vi.mock("shaders/react", watch("shaders/react"));

// Importing the shell's whole module graph can pass vitest's 5s default under full-suite load.
const shellImport = 30_000;

describe("cold start graph", () => {
  it("loads the shell without any page, the rich text editor, the canvas, voice or the shader", async () => {
    await import("@/App");
    expect([...evaluated]).toEqual([]);
  }, shellImport);
});
