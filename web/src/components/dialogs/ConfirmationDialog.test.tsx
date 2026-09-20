import type { ReactElement } from "react";
import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { describe, expect, it } from "vitest";

import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";

const ConfirmationHarness = () => {
  const { open } = useConfirmationDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => setResult(String(await open({ message: "Delete this service?" })))}
      >
        Ask
      </button>
      <p>{result}</p>
    </div>
  );
};

const renderWithRoot = (ui: ReactElement) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <ContextAwareConfirmation.ConfirmationRoot />
      {ui}
    </QueryClientProvider>,
  );

describe("useConfirmationDialog", () => {
  it("resolves true when the confirm button is clicked", async () => {
    const user = userEvent.setup();
    renderWithRoot(<ConfirmationHarness />);

    await user.click(screen.getByRole("button", { name: "Ask" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));

    expect(await screen.findByText("true")).toBeInTheDocument();
  });

  it("resolves false when the cancel button is clicked", async () => {
    const user = userEvent.setup();
    renderWithRoot(<ConfirmationHarness />);

    await user.click(screen.getByRole("button", { name: "Ask" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(await screen.findByText("false")).toBeInTheDocument();
  });

  it("resolves false on Escape", async () => {
    const user = userEvent.setup();
    renderWithRoot(<ConfirmationHarness />);

    await user.click(screen.getByRole("button", { name: "Ask" }));
    await user.keyboard("{Escape}");

    expect(await screen.findByText("false")).toBeInTheDocument();
  });

  it("resolves false on a backdrop click", async () => {
    const user = userEvent.setup();
    renderWithRoot(<ConfirmationHarness />);

    await user.click(screen.getByRole("button", { name: "Ask" }));
    const overlay = document.querySelector("[data-slot='dialog-overlay']");
    expect(overlay).not.toBeNull();
    await user.click(overlay as Element);

    expect(await screen.findByText("false")).toBeInTheDocument();
  });

  it("uses the message as the heading when no title is provided", async () => {
    const user = userEvent.setup();
    renderWithRoot(<ConfirmationHarness />);

    await user.click(screen.getByRole("button", { name: "Ask" }));

    expect(screen.getByRole("heading", { name: "Delete this service?" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("renders the title with the message as description when a title is provided", async () => {
    const user = userEvent.setup();
    renderWithRoot(
      <ConfirmationHarnessWithOptions options={{ message: "Delete?", title: "Delete service" }} />,
    );

    await user.click(screen.getByRole("button", { name: "Ask" }));

    expect(screen.getByRole("heading", { name: "Delete service" })).toBeInTheDocument();
    expect(screen.getByText("Delete?")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("renders custom labels when provided", async () => {
    const user = userEvent.setup();
    renderWithRoot(
      <ConfirmationHarnessWithOptions
        options={{ message: "Delete?", confirmLabel: "Delete it", cancelLabel: "Keep it" }}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Ask" }));

    expect(screen.getByRole("button", { name: "Delete it" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Keep it" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("defaults the confirm button to the destructive variant", async () => {
    const user = userEvent.setup();
    renderWithRoot(<ConfirmationHarness />);

    await user.click(screen.getByRole("button", { name: "Ask" }));

    expect(screen.getByRole("button", { name: "Confirm" }).className).toContain("bg-destructive");
    await user.keyboard("{Escape}");
  });

  it("uses the default variant when destructive is false", async () => {
    const user = userEvent.setup();
    renderWithRoot(
      <ConfirmationHarnessWithOptions options={{ message: "Archive?", destructive: false }} />,
    );

    await user.click(screen.getByRole("button", { name: "Ask" }));

    expect(screen.getByRole("button", { name: "Confirm" }).className).not.toContain(
      "bg-destructive",
    );
    await user.keyboard("{Escape}");
  });
});

interface ConfirmationHarnessWithOptionsProps {
  options: {
    message: string;
    title?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    destructive?: boolean;
  };
}

const ConfirmationHarnessWithOptions = ({
  options,
}: ConfirmationHarnessWithOptionsProps) => {
  const { open } = useConfirmationDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button type="button" onClick={async () => setResult(String(await open(options)))}>
        Ask
      </button>
      <p>{result}</p>
    </div>
  );
};
