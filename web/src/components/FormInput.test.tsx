import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { describe, expect, it } from "vitest";

import { FormInput } from "@/components/FormInput";

const Harness = () => {
  const form = useForm<{ title: string }>({ defaultValues: { title: "" } });
  return (
    <form>
      <FormInput control={form.control} name="title" label="Title" />
      <button type="button" onClick={() => form.setError("title", { message: "Title is required" })}>
        Trigger
      </button>
    </form>
  );
};

const PlaceholderHarness = () => {
  const form = useForm<{ email: string }>({ defaultValues: { email: "" } });
  return (
    <form>
      <FormInput
        control={form.control}
        name="email"
        label="Email"
        placeholder="you@example.com"
      />
    </form>
  );
};

describe("FormInput", () => {
  it("renders a labelled text input", () => {
    render(<Harness />);
    expect(screen.getByLabelText("Title")).toBeInTheDocument();
  });

  it("passes through extra input props", () => {
    render(<PlaceholderHarness />);
    expect(screen.getByPlaceholderText("you@example.com")).toBeInTheDocument();
  });

  it("shows the field error when present", async () => {
    const user = userEvent.setup();
    render(<Harness />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Trigger" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Title is required");
  });
});
