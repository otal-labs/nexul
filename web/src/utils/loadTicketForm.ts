// The ticket form carries the rich-text editor, so it loads when a ticket dialog first opens, never with the pages that offer one.
export const loadTicketForm = async () => {
  const [form, footer] = await Promise.all([import("@/components/ticket/CreateTicketForm"), import("@/components/ticket/CreateTicketFooter")]);
  return { CreateTicketForm: form.CreateTicketForm, CreateTicketFooter: footer.CreateTicketFooter };
};
