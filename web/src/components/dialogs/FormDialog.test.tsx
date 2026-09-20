import type { ComponentType, ReactElement } from "react";
import { useEffect, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { useFormDialog } from "@/hooks/useFormDialog";

const TestFormSchema = z.object({
  title: z.string().min(3, "Title is required"),
});
type TestFormData = z.infer<typeof TestFormSchema>;

const TestForm = ({ fail = false }: { fail?: boolean }) => {
  const { register, formState, onSubmit } = useFormDialogContext<TestFormData>();
  onSubmit(async (data) => {
    if (fail) throw new Error("boom");
    return data;
  });
  return (
    <>
      <input placeholder="Title" {...register("title")} />
      {formState.errors.title && (
        <p role="alert">{formState.errors.title.message}</p>
      )}
    </>
  );
};

const NoopForm = () => {
  const { register } = useFormDialogContext<TestFormData>();
  return <input placeholder="Title" {...register("title")} />;
};

const LoadingForm = () => {
  const { register, setLoading } = useFormDialogContext<TestFormData>();
  useEffect(() => {
    setLoading(true);
  }, [setLoading]);
  return <input placeholder="Title" {...register("title")} />;
};

// Exercises the "Create more"-style stay-open contract: flags via context that a successful
// submit should keep the dialog open instead of resolving it, then registers the cleanup that
// runs in its place (mirrors CreateTicketFooter's real usage).
const StayOpenForm = () => {
  const { register, onSubmit, setStayOpen, onAfterSubmit } = useFormDialogContext<TestFormData>();
  const [cleanedUp, setCleanedUp] = useState(false);
  useEffect(() => {
    setStayOpen(true);
    onAfterSubmit(() => setCleanedUp(true));
  }, [setStayOpen, onAfterSubmit]);
  onSubmit(async (data) => data);
  return (
    <>
      <input placeholder="Title" {...register("title")} />
      {cleanedUp && <p>cleaned up</p>}
    </>
  );
};

const formatResult = (result: { success: boolean; data: TestFormData | null }): string => {
  if (!result.success) return "cancelled";
  return JSON.stringify(result.data);
};

interface FormHarnessProps {
  form: ReactElement | ComponentType;
  okLabel?: string;
  header?: ReactElement;
  footerStart?: ReactElement;
}

const FormHarness = ({ form, okLabel = "Save", header, footerStart }: FormHarnessProps) => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<TestFormData>({
            title: "New ticket",
            description: "Fill the form",
            schema: TestFormSchema,
            okLabel,
            header,
            footerStart,
            form,
          });
          setResult(formatResult(result));
        }}
      >
        Open
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

describe("useFormDialog", () => {
  it("renders the title and description", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));

    expect(screen.getByText("New ticket")).toBeInTheDocument();
    expect(screen.getByText("Fill the form")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("resolves the form data when the submit handler succeeds", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByPlaceholderText("Title"), "New ticket");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText('{"title":"New ticket"}')).toBeInTheDocument();
  });

  it("resolves the raw data when the form registers no submit handler", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<NoopForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByPlaceholderText("Title"), "New ticket");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText('{"title":"New ticket"}')).toBeInTheDocument();
  });

  it("keeps the dialog open on validation failure", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByPlaceholderText("Title"), "ab");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(screen.getByText("Title is required")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("pending")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("keeps the dialog open and surfaces the error when the handler throws", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm fail />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByPlaceholderText("Title"), "New ticket");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("boom")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("pending")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("resolves cancelled when the cancel button is clicked", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(await screen.findByText("cancelled")).toBeInTheDocument();
  });

  it("resolves cancelled on Escape", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.keyboard("{Escape}");

    expect(await screen.findByText("cancelled")).toBeInTheDocument();
  });

  it("resolves cancelled on a backdrop click", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    const overlay = document.querySelector("[data-slot='dialog-overlay']");
    expect(overlay).not.toBeNull();
    await user.click(overlay as Element);

    expect(await screen.findByText("cancelled")).toBeInTheDocument();
  });

  it("renders a form passed as a component type", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={TestForm} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByPlaceholderText("Title"), "New ticket");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText('{"title":"New ticket"}')).toBeInTheDocument();
  });

  it("disables the ok button while setLoading is active", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<LoadingForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    await user.keyboard("{Escape}");
  });

  it("renders a custom ok label", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} okLabel="Create" />);

    await user.click(screen.getByRole("button", { name: "Open" }));

    expect(screen.getByRole("button", { name: "Create" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("throws when useFormDialogContext is used outside a FormDialog", () => {
    expect(() => {
      render(<TestForm />);
    }).toThrow("useFormDialogContext must be used within a FormDialog");
  });

  it("renders a header in place of the title, keeping the title as the dialog's accessible name", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} header={<div>Custom header</div>} />);

    await user.click(screen.getByRole("button", { name: "Open" }));

    expect(screen.getByText("Custom header")).toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: "New ticket" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("renders footerStart at the left of the footer, alongside Cancel/OK", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<TestForm />} footerStart={<span>Left content</span>} />);

    await user.click(screen.getByRole("button", { name: "Open" }));

    expect(screen.getByText("Left content")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("keeps the dialog open and runs the registered cleanup when the form flags stay-open", async () => {
    const user = userEvent.setup();
    renderWithRoot(<FormHarness form={<StayOpenForm />} />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByPlaceholderText("Title"), "New ticket");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("cleaned up")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("pending")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });
});
