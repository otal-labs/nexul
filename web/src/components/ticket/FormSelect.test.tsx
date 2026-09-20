import { zodResolver } from "@hookform/resolvers/zod";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { FormSelect } from "@/components/ticket/FormSelect";

const TestSchema = z.object({
  project_id: z.string().min(1, "A project is required"),
});

type TestFormData = z.infer<typeof TestSchema>;

const options = [
  { value: "p-1", label: "Backend" },
  { value: "p-2", label: "Frontend" },
];

const Wrapper = ({
  onSubmit,
  placeholder,
}: {
  onSubmit: (data: TestFormData) => void;
  placeholder?: string;
}) => {
  const form = useForm<TestFormData>({
    defaultValues: { project_id: "" },
    resolver: zodResolver(TestSchema),
  });
  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <FormSelect
          control={form.control}
          name="project_id"
          label="Project"
          options={options}
          {...(placeholder ? { placeholder } : {})}
        />
        <button type="submit">Save</button>
      </form>
    </FormProvider>
  );
};

describe("FormSelect", () => {
  it("renders the label and, once opened, every option", async () => {
    const user = userEvent.setup();
    render(<Wrapper onSubmit={() => {}} />);
    await user.click(screen.getByRole("combobox", { name: "Project" }));
    expect(await screen.findByRole("option", { name: "Backend" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Frontend" })).toBeInTheDocument();
  });

  it("renders a placeholder option when provided", async () => {
    const user = userEvent.setup();
    render(<Wrapper onSubmit={() => {}} placeholder="Uncategorized" />);
    await user.click(screen.getByRole("combobox", { name: "Project" }));
    expect(await screen.findByRole("option", { name: "Uncategorized" })).toBeInTheDocument();
  });

  it("submits the selected value", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<Wrapper onSubmit={onSubmit} />);

    await user.click(screen.getByRole("combobox", { name: "Project" }));
    await user.click(await screen.findByRole("option", { name: "Frontend" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ project_id: "p-2" }), expect.anything());
  });

  it("surfaces the validation error when nothing is selected", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<Wrapper onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(screen.getByText("A project is required")).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
