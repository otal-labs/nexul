# 13 — Push notifications

**Type:** grilling
**Status:** resolved
**Blocked by:** 02

## Question

What reaches the phone as a push notification, and how? Which events notify (inbox items only, or deploy results too), how the push token attaches to the phone's device session so signing out a device stops its pushes, the id-only payload shape, what happens on tap, and the one-time Expo and Firebase project setup the owner does.

## Answer

Decided 2026-09-28.

- **What notifies:** exactly what lands in a user's Inbox (the notifications
  table), nothing else, so the phone and the Inbox never disagree.
- **Registration:** the app registers its Expo push token on its own session
  (`PUT /api/auth/sessions/current/push-token`, stored on the sessions row). A
  signed-out or expired session stops pushing because the row is gone.
- **Sending:** when notifications are created for a user, the server posts one
  message per phone session with a push token to Expo's push API. Failures
  are logged, never retried in a loop. A `DeviceNotRegistered` receipt
  clears that token.
- **Payload:** title "Nexul", body "New activity in <workspace name>", data
  `{notification_id, host}`. Tapping opens the app on that Inbox item, which
  fetches the content from the instance.
- **Owner's one-time setup:** an Expo account and project (its `projectId`
  goes into the app config), plus a Firebase project whose FCM credentials are
  uploaded to Expo. Nothing for self-hosters.
