# Runs at once belongs to the computer

Amends ADR 0132's runs-at-once amendment.

A workspace set how many runs a person could have going, 1 to 5, counted across every workspace because all of a
person's runs shared their one computer. With runs failing over across several computers (ADR 0165) the limit was
on the wrong thing: two computers gave the same five slots as one, and a workspace Owner was deciding how hard
someone's own machine works.

Decision: each paired computer has its own **runs at once**, 1 to 5, set by its owner.

- **What counts.** The play runs going on that computer, pressed or automatic, whoever started them, so a run landed
  through a share takes room too. `@Agent` chats do not count.
- **Who waits.** A queued auto run starts on a computer with room and waits when none has any. A pressed run uses room
  to choose but never waits.
- **The workspace setting is removed**, from the auto plays panel, `workspace_update` and the limits route. The
  `auto_play.limits_updated` event keeps the field in its contract, deprecated, and no longer sends it as a choice.
- **On upgrade** each computer takes the highest value its owner had in any workspace, so nobody loses room.

The trade-off: a workspace can no longer stop one member's runs from crowding a shared queue. The daily cap per
ticket still bounds a loop, and nothing a workspace pays for is spent by a member's own computers.

Rejected: keeping the workspace number as a ceiling on top, which brings back the wall the change removes; a per-person
total beside the per-computer one, which is two numbers to explain for one question.

Decided 2026-10-22.
