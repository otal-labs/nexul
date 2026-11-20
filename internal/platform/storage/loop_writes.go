package storage

// The writes that can give each background loop work, for CommitsTo: every table its pass reads, except where noted.
var (
	// OutboxRelayWrites includes the relay's own mark-published, which wakes it for the next batch of a backlog.
	OutboxRelayWrites = []TableWrite{Writes("outbox")}
	// WebhookRelayWrites includes the relay's own delivery marks, which wake it for the next batch of a backlog.
	WebhookRelayWrites = []TableWrite{Writes("integration_deliveries")}
	// DialinWrites leaves out automation_cursors, which only ever advance, and outbox updates and deletes, which never
	// add an event past a cursor.
	DialinWrites = []TableWrite{Inserts("outbox"), Writes("automations"), Writes("workspaces")}
	SettleWrites = []TableWrite{Writes("doc_settles")}
	// IssueSweepWrites wakes the issue sweep to backfill a sync just turned on; the sweep itself runs on a timer.
	IssueSweepWrites = []TableWrite{Writes("github_issue_sync")}
	// ImportWrites includes the loop's own progress writes, which wake it again after a run for nothing new.
	ImportWrites = []TableWrite{Writes("imports")}
	// IssuePushWrites wakes the write-back for a push queued, deferred or done; the next due wakes it by timer.
	IssuePushWrites = []TableWrite{Writes("github_issue_pushes"), Writes("github_comment_pushes")}
	ScheduleWrites  = []TableWrite{Writes("scheduled_messages")}
	ReminderWrites  = []TableWrite{
		Writes("meetings"), Writes("meeting_invitees"), Writes("tickets"), Writes("projects"), Writes("users"),
		Writes("statuses"), Writes("ticket_due_reminders"),
	}
)
