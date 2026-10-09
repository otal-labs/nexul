import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InterviewSourceAddForm } from "@/components/memory/InterviewSourceAddForm";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: (e: Error) => e.message }));

const onClose = vi.fn();

const renderForm = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <InterviewSourceAddForm projectId="p-1" onClose={onClose} />
    </QueryClientProvider>,
  );

const stanceOn = (name: "Follow" | "Question") => screen.getByRole("radio", { name }).getAttribute("data-state") === "on";

beforeEach(() => {
  mocks.get.mockReset().mockResolvedValue({ data: [] });
  mocks.post.mockReset().mockResolvedValue({ data: { id: "s-1", project_id: "p-1" } });
  onClose.mockReset();
});

describe("InterviewSourceAddForm", () => {
  it("preselects follow for markdown and folders, question for code and projects", async () => {
    const user = userEvent.setup();
    renderForm();
    const path = screen.getByRole("textbox", { name: "Path" });
    await user.type(path, "cmd/main.go");
    expect(stanceOn("Question")).toBe(true);
    await user.clear(path);
    await user.type(path, "practices/");
    expect(stanceOn("Follow")).toBe(true);
    await user.click(screen.getByRole("radio", { name: "Project" }));
    expect(stanceOn("Question")).toBe(true);
    await user.click(screen.getByRole("radio", { name: "Paste text" }));
    expect(stanceOn("Follow")).toBe(true);
  });

  it("keeps a stance the person picked when the path changes", async () => {
    const user = userEvent.setup();
    renderForm();
    await user.click(screen.getByRole("radio", { name: "Question" }));
    await user.type(screen.getByRole("textbox", { name: "Path" }), "docs/rules.md");
    await user.click(screen.getByRole("button", { name: "Add" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/memories/interview-sources", {
      project_id: "p-1", kind: "path", ref: "docs/rules.md", label: "", body: "", stance: "question",
    });
    expect(onClose).toHaveBeenCalled();
  });

  it("refuses an empty path without posting", async () => {
    const user = userEvent.setup();
    renderForm();
    await user.click(screen.getByRole("button", { name: "Add" }));
    expect(await screen.findByText("Enter a path")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("refuses a file that isn't text or markdown", async () => {
    const user = userEvent.setup({ applyAccept: false });
    renderForm();
    await user.click(screen.getByRole("radio", { name: "Paste text" }));
    await user.upload(screen.getByLabelText("or drop a text or markdown file"), new File(["%PDF"], "spec.pdf", { type: "application/pdf" }));
    expect(await screen.findByText("spec.pdf isn't text or markdown. Paste its text instead.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add" }));
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("reads a markdown file as pasted text labelled with its name", async () => {
    const user = userEvent.setup();
    renderForm();
    await user.click(screen.getByRole("radio", { name: "Paste text" }));
    await user.upload(screen.getByLabelText("or drop a text or markdown file"), new File(["# Rules\nNo else."], "rules.md", { type: "text/markdown" }));
    expect(await screen.findByDisplayValue("rules.md")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/memories/interview-sources", {
      project_id: "p-1", kind: "text", ref: "", label: "rules.md", body: "# Rules\nNo else.", stance: "follow",
    });
  });
});
