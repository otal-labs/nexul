import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { describe, expect, it } from "vitest";

import { FormCombobox } from "@/components/FormCombobox";

const OPTIONS = [
  { value: "p1", label: "StreamerBotChat" },
  { value: "p2", label: "keystrokeacademy" },
  { value: "p3", label: "Nexul" },
];

const Harness = () => {
  const form = useForm<{ project: string }>({ defaultValues: { project: "" } });
  return (
    <div>
      <FormCombobox control={form.control} name="project" label="Project" placeholder="Pick a project" options={OPTIONS} />
      <output data-testid="value">{form.watch("project")}</output>
    </div>
  );
};

describe("FormCombobox", () => {
  it("filters options as you type and picks with Enter", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByRole("combobox"));
    await user.keyboard("nex");

    expect(screen.getByRole("option", { name: "Nexul" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "StreamerBotChat" })).not.toBeInTheDocument();

    await user.keyboard("{Enter}");
    expect(screen.getByTestId("value")).toHaveTextContent("p3");
    expect(screen.getByRole("combobox")).toHaveTextContent("Nexul");
  });

  it("picks by click and clears via the placeholder row", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByRole("option", { name: "keystrokeacademy" }));
    expect(screen.getByTestId("value")).toHaveTextContent("p2");

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByRole("option", { name: "Pick a project" }));
    expect(screen.getByTestId("value")).toHaveTextContent("");
  });

  it("shows a no-matches row for a filter that hits nothing", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByRole("combobox"));
    await user.keyboard("zzz");
    expect(screen.getByText("No matches.")).toBeInTheDocument();
  });
});
