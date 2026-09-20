import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { QuestionCard } from "@/components/play/QuestionCard";
import type { HarnessQuestion } from "@/models/Question";

const twoQuestions: HarnessQuestion = {
  request_id: "req-1",
  questions: [
    {
      id: "q1",
      text: "How do you want to proceed?",
      header: "Nexul MCP down",
      options: [
        { label: "Proceed without Nexul tools", description: "Skip the ticket links." },
        { label: "Stop and wait", description: "Do nothing yet." },
      ],
    },
    { id: "q2", text: "Which areas?", options: [{ label: "Docs" }, { label: "Tests" }], multi_select: true },
  ],
};

const openEnded: HarnessQuestion = { request_id: "req-2", questions: [{ id: "name", text: "What name?", options: [] }] };

describe("QuestionCard", () => {
  it("shows the progress line, the question as title, its hint muted, and one numbered row per option", () => {
    render(<QuestionCard question={twoQuestions} onSubmit={vi.fn()} />);
    expect(screen.getByText("Question 1 of 2")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "How do you want to proceed?" })).toBeInTheDocument();
    expect(screen.getByText("Nexul MCP down")).toBeInTheDocument();
    expect(screen.getAllByRole("radio")).toHaveLength(2);
    expect(screen.getByText("Skip the ticket links.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
  });

  it("number keys pick an option, Enter moves on, and the last step sends every answer", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<QuestionCard question={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("group", { name: "Question from the Agent" }));
    await user.keyboard("2");
    expect(screen.getByRole("radio", { name: "Stop and wait" })).toBeChecked();
    await user.keyboard("{Enter}");

    expect(screen.getByText("Question 2 of 2")).toBeInTheDocument();
    expect(screen.getAllByRole("checkbox")).toHaveLength(2);
    await user.keyboard("1");
    await user.keyboard("2");
    expect(screen.getByRole("checkbox", { name: "Docs" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Tests" })).toBeChecked();
    await user.click(screen.getByRole("button", { name: "Send answer" }));

    expect(onSubmit).toHaveBeenCalledWith({ q1: { selected: ["Stop and wait"] }, q2: { selected: ["Docs", "Tests"] } });
  });

  it("typing something else replaces the picked option, and Back returns to the previous question", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<QuestionCard question={twoQuestions} onSubmit={onSubmit} />);

    await user.click(screen.getByRole("radio", { name: "Proceed without Nexul tools" }));
    await user.type(screen.getByRole("textbox", { name: "Something else" }), "Ask Onik first");
    expect(screen.getByRole("radio", { name: "Proceed without Nexul tools" })).not.toBeChecked();
    await user.click(screen.getByRole("button", { name: "Next" }));
    await user.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByText("Question 1 of 2")).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Something else" })).toHaveValue("Ask Onik first");
  });

  it("a question without options is a free-text field, and Enter in it sends", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<QuestionCard question={openEnded} onSubmit={onSubmit} />);
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
    await user.type(screen.getByRole("textbox", { name: "Your answer" }), "Bot{Enter}");
    expect(onSubmit).toHaveBeenCalledWith({ name: { text: "Bot" } });
  });

  it("with an answer given it is read-only and shows what was answered", () => {
    render(<QuestionCard question={twoQuestions} answer={{ answers: { q1: { selected: ["Stop and wait"] }, q2: { text: "Neither" } } }} />);
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Send answer|Next/ })).not.toBeInTheDocument();
    expect(screen.getByText("→ Stop and wait")).toBeInTheDocument();
    expect(screen.getByText("→ Neither")).toBeInTheDocument();
  });
});
