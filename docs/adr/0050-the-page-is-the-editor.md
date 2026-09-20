# The page is the editor: no dialogs, no read/edit toggle, no Save button

Docs and tickets are edited in place on the page that displays them. The title
and body are always live inputs; formatting appears in a bubble over a text
selection; writes autosave on a ~800ms debounce with a flush on blur and a
quiet Saved/Saving indicator in the meta row. There is no Edit button, no
read mode, no Save button, and no modal editing surface.

This was arrived at by rejection, not by preference, which is why it is
recorded: the doc body first shipped as a bordered WYSIWYG editor with a fixed
toolbar behind an **Edit** button, and the owner's verdict was "why would I
need a WYSIWYG editor? Just edit the docs immediately." The box, the toolbar
and the toggle were removed, and when ticket editing was built it replicated
the doc surface rather than reintroducing the click-to-edit toggle with
Save/Cancel buttons that had shipped for tickets in the meantime.

The trade-off taken with autosave: there is no explicit commit point, so every
editable surface owes an error path — the mutation hook surfaces a failed save
as a toast — and a create form may still use a plain field, because plain text
round-trips through the legacy path and upgrades to the structured body on the
first edit. The create-ticket dialog stays a plain textarea for a second reason
that outlives the legacy path: an attachment is owned by exactly one existing
ticket (ADR 0027), so there is nothing to attach a pasted image to until the
ticket has been created, and the full editor would offer a paste target that
cannot work. The cost of reversing this is every editable surface in the
product, so a new one follows the pattern rather than inventing its own.
