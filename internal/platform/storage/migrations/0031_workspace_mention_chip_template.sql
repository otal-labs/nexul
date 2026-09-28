-- The mention chip layout belongs to the workspace whose tickets it renders, not to the instance.
ALTER TABLE workspaces ADD COLUMN mention_chip_template TEXT NOT NULL DEFAULT '{ticket.Ticket} {ticket.Status}';
ALTER TABLE instance_settings DROP COLUMN mention_chip_template;
