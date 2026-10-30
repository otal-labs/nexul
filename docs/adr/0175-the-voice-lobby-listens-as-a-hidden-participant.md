# The voice lobby listens as a hidden participant

The lobby row above a voice channel's chat shows who is in the call before you join. Showing who is speaking
there as well needs LiveKit's speaker updates, which arrive over a room connection and which no REST call returns.

Decision: while the lobby shows the join prompt for a room with people in it, the browser connects one extra
participant to that room, the **listener**, through `POST /api/voice/{conversationID}/listen`.

- **Nobody sees it.** The token's identity is `listen:<user id>`, never the person's own, and its grant is `hidden`:
  LiveKit does not list a hidden participant to the others. It cannot publish tracks or data, and it cannot create
  the room, so it never starts a call. The route runs the same checks as joining: a voice channel, a configured
  LiveKit connector, and the caller's read of the channel through chat's own rule.
- **It is never presence.** Occupancy refuses every `listen:` identity at the one place all three feeds write
  (the join, webhook and reconciliation paths share the store), so a listener never appears in the occupant list,
  the count, the `voice.occupancy.changed` event, or the "alone for five minutes" rule. Removing someone from a
  private channel's call removes their listener too, so leaving a private call cannot leave a connection behind.
- **What it receives.** LiveKit sends a speaker update only to a participant subscribed to the speaker (its
  `SendSpeakerUpdate` skips any speaker the participant has no subscription to), so a participant with
  `canSubscribe: false`, or one that never subscribes, hears nothing about who is speaking. The listener therefore
  keeps `canSubscribe: true`, connects with `autoSubscribe: false`, subscribes to audio tracks only and disables each
  one in the same step, so the server forwards no audio. It never subscribes to video or screen share. The
  speaker updates are the only thing it uses.
- **When it runs.** One listener at a time, only on the join prompt of a room that has people, disconnected the
  moment the lobby goes away, you join, or the room empties. A failed listen (no connector, a network failure)
  shows no rings and no error.

The cost: a person looking at a busy lobby holds one more LiveKit connection, and a room stays open while a
listener is the only one left, until the browser sees the room empty and disconnects it. The listener's token lasts
five minutes: it is presented once, to connect, and LiveKit refreshes a connected participant's token itself. A
member removed from a private channel is disconnected at once, and can reconnect a listener for only those five
minutes, which would show who is speaking, never what is said.

Decided 2026-10-30.
