# Discord execute-webhook contract

All facts below come from Discord's own developer documentation at
discord.com/developers/docs (mirrored source: the `discord/discord-api-docs`
repository, `main` branch, files `developers/resources/webhook.mdx`,
`developers/resources/message.mdx`, `developers/topics/rate-limits.mdx`,
`developers/reference.mdx`, `developers/topics/opcodes-and-status-codes.mdx`),
plus GitHub's own webhook docs and the published source of Grafana's and
Uptime Kuma's Discord notifiers. Each claim is followed by its source URL.

**The five facts a compatible endpoint most needs:**

1. A webhook URL is `POST /webhooks/{webhook.id}/{webhook.token}`. The id is
   just the webhook's own snowflake (not meaningful to a sender beyond
   identifying which webhook), and the token is an opaque secret with no
   documented fixed length or alphabet.
2. A request must set at least one of `content`, `embeds`, `components`,
   `file`, or `poll` (the docs give two slightly different "required" lists,
   quoted below). `content` caps at 2000 characters, up to 10 embeds per
   message, 25 fields per embed, and a combined 6000-character cap across all
   text fields of all embeds in the message.
3. `wait=false` (the default on the plain Execute Webhook route) returns
   `204 No Content`; `wait=true` returns the created message body. The two
   compatibility routes (`/slack`, `/github`) default `wait` to `true`
   instead.
4. Files go over `multipart/form-data`, with non-file fields collapsed into
   a single `payload_json` form field and each file as `files[n]`; embeds can
   reference an uploaded file via `attachment://filename`.
5. Rate limiting is bucketed per `webhook_id + webhook_token` like any other
   Discord route (headers `X-RateLimit-*`, HTTP 429 with a JSON body carrying
   `retry_after`); the docs do not publish a fixed numeric quota for the
   webhook-execute bucket specifically, only the global 50 requests/second
   cap and the general 429/`Retry-After` mechanism.

## URL shape, token, and id

The webhook resource object has an `id` (snowflake) and a `token` (string,
"the secure token of the webhook, returned for Incoming Webhooks")
(https://discord.com/developers/docs/resources/webhook#webhook-object-webhook-structure).

Execute Webhook is `POST /webhooks/{webhook.id}/{webhook.token}`
(https://discord.com/developers/docs/resources/webhook#execute-webhook).

The docs do not state a fixed token length or character-set contract. The
one example token shown in the "Example Incoming Webhook" object is:

```
3d89bb7572e0fb30d8128367b3b1b44fecd1726de135cbe28a41f8b2f777c372ba2939e72279b94526ff5d1bd4358d65cf11
```

— 100 characters, hex digits only in this particular example
(https://discord.com/developers/docs/resources/webhook#webhook-object-example-incoming-webhook).
Treating that length/alphabet as a guarantee is **unverified**: it is one
example value, not a stated spec.

The `id` is the webhook's own snowflake, i.e. it identifies which webhook
object this is; it carries no other meaning to a sender beyond that (guild,
channel, and application association are separate fields on the resource,
not encoded in a way a sender needs to parse)
(https://discord.com/developers/docs/resources/webhook#webhook-object-webhook-structure).
Per the general snowflake spec, the id does embed a creation timestamp in
its top 42 bits if a sender wanted to decode one, but the webhook resource
doc does not call this out as a supported use
(https://discord.com/developers/docs/reference#snowflakes).

## Execute Webhook: every request field, required-ness, and limits

Two places in the docs describe "required":

> Note that when sending a message, you must provide a value for at
> **least one of** `content`, `embeds`, `components`, `file`, or `poll`.

(https://discord.com/developers/docs/resources/webhook#execute-webhook)

The JSON/Form Params table instead marks the "one of" group as `content`,
`file`, `embeds`, `poll` only — `components` is listed as not required in
that table. Both are quoted verbatim below; the docs do not reconcile the
discrepancy.

JSON/Form Params table, verbatim
(https://discord.com/developers/docs/resources/webhook#execute-webhook-jsonform-params):

| Field | Type | Description | Required |
|---|---|---|---|
| content | string | the message contents (up to 2000 characters) | one of content, file, embeds, poll |
| username | string | override the default username of the webhook | false |
| avatar_url | string | override the default avatar of the webhook | false |
| tts | boolean | true if this is a TTS message | false |
| embeds | array of up to 10 embed objects | embedded `rich` content | one of content, file, embeds, poll |
| allowed_mentions | allowed mention object | allowed mentions for the message | false |
| components * | array of message component | the components to include with the message | false |
| files[n] ** | file contents | the contents of the file being sent | one of content, file, embeds, poll |
| payload_json ** | string | JSON encoded body of non-file params | `multipart/form-data` only |
| attachments ** | array of partial attachment request objects | metadata for the attachments | false |
| flags *** | integer | message flags combined as a bitfield (only `SUPPRESS_EMBEDS`, `SUPPRESS_NOTIFICATIONS` and `IS_COMPONENTS_V2` can be set) | false |
| thread_name | string | name of thread to create (requires the webhook channel to be a forum or media channel) | false |
| applied_tags | array of snowflakes | array of tag ids to apply to the thread (requires the webhook channel to be a forum or media channel) | false |
| poll | poll request object | A poll! | one of content, file, embeds, poll |

Notes attached to that table (verbatim, condensed):

- `*` "Application-owned webhooks can always send components. Non-application
  -owned webhooks cannot send interactive components, and the `components`
  field will be ignored unless they set the `with_components` query param."
- `**` "See Uploading Files for details."
- `***` "When the flag `IS_COMPONENTS_V2` is set, the webhook message can
  only contain `components`. Providing `content`, `embeds`, `files[n]` or
  `poll` will fail with a 400 BAD REQUEST response."

There is also a warning worth carrying into a compatible implementation:

> Discord may strip certain characters from message content, like invalid
> unicode characters or characters which cause unexpected message
> formatting. If you are passing user-generated strings into message
> content, consider sanitizing the data to prevent unexpected behavior and
> using `allowed_mentions` to prevent unexpected mentions.

(https://discord.com/developers/docs/resources/webhook#execute-webhook)

For embed objects specifically, the webhook doc adds a constraint not in
the general embed object doc:

> For the webhook embed objects, you can set every field except `type` (it
> will be `rich` regardless of if you try to set it), `provider`, `video`,
> and any `height`, `width`, or `proxy_url` values for images.

(https://discord.com/developers/docs/resources/webhook#execute-webhook)

`allowed_mentions` is documented on the Message resource, not the webhook
resource (see next section for its structure)
(https://discord.com/developers/docs/resources/message#allowed-mentions-object).

## The embed object

Embed Structure, verbatim
(https://discord.com/developers/docs/resources/message#embed-object-embed-structure):

| Field | Type | Description |
|---|---|---|
| title? | string | title of embed |
| type? | string | type of embed (always "rich" for webhook embeds) |
| description? | string | description of embed |
| url? | string | url of embed |
| timestamp? | ISO8601 timestamp | timestamp of embed content |
| color? | integer | color code of the embed |
| footer? | embed footer object | footer information |
| image? | embed image object | image information |
| thumbnail? | embed image object | thumbnail information |
| video? | embed video object | video information |
| provider? | embed provider object | provider information |
| author? | embed author object | author information |
| fields? | array of embed field objects | fields information, max of 25 |
| flags? | integer | embed flags combined as a bitfield |

Footer: `text` (string, required), `icon_url?` (http(s) or attachment),
`proxy_icon_url?`.
Image / thumbnail: `url` (required, "only supports http(s) and
attachments"), `proxy_url?`, `height?`, `width?`, `content_type?`,
`placeholder?`, `placeholder_version?`, `description?`, `flags?`.
Author: `name` (required), `url?` (http(s) only), `icon_url?` (http(s) or
attachment), `proxy_icon_url?`.
Field: `name` (required), `value` (required), `inline?` (boolean)
(https://discord.com/developers/docs/resources/message#embed-object-embed-footer-structure,
https://discord.com/developers/docs/resources/message#embed-object-embed-image-structure,
https://discord.com/developers/docs/resources/message#embed-object-embed-author-structure,
https://discord.com/developers/docs/resources/message#embed-object-embed-field-structure).

Embed Limits, verbatim
(https://discord.com/developers/docs/resources/message#embed-object-embed-limits):

> All of the following limits are measured inclusively. Leading and
> trailing whitespace characters are not included (they are trimmed
> automatically).

| Field | Limit |
|---|---|
| title | 256 characters |
| description | 4096 characters |
| fields | Up to 25 field objects |
| field.name | 256 characters |
| field.value | 1024 characters |
| footer.text | 2048 characters |
| author.name | 256 characters |

> Additionally, the combined sum of characters in all `title`,
> `description`, `field.name`, `field.value`, `footer.text`, and
> `author.name` fields across all embeds attached to a message must not
> exceed 6000 characters. Violating any of these constraints will result
> in a `Bad Request` response.
>
> Embeds are deduplicated by URL. If a message contains multiple embeds
> with the same URL, only the first is shown.

(same URL as above)

## Query parameters `wait` and `thread_id`, and the response shape

Execute Webhook query params, verbatim
(https://discord.com/developers/docs/resources/webhook#execute-webhook-query-string-params):

| Field | Type | Description | Required |
|---|---|---|---|
| wait | boolean | waits for server confirmation of message send before response, and returns the created message body (defaults to `false`; when `false` a message that is not saved does not return an error) | false |
| thread_id | snowflake | Send a message to the specified thread within a webhook's channel. The thread will automatically be unarchived. | false |
| with_components | boolean | whether to respect the `components` field of the request... (defaults to `false`) | false |

The endpoint summary states: "Returns a message or `204 No Content`
depending on the `wait` query parameter."
(https://discord.com/developers/docs/resources/webhook#execute-webhook).
So: `wait=false` (default) -> `204 No Content`; `wait=true` -> a message
object body. The docs do not print an explicit numeric status code for the
`wait=true` case on this specific route; the general HTTP Response Codes
table lists `200 (OK)` as "the request completed successfully"
(https://discord.com/developers/docs/topics/opcodes-and-status-codes#http-http-response-codes),
so 200 is the reasonable read but not spelled out verbatim for this route —
flagging as **inferred, not directly stated**.

For the Slack- and GitHub-compatible routes, `wait` defaults to `true`
instead of `false`:

> wait — waits for server confirmation of message send before response
> (defaults to `true`; when `false` a message that is not saved does not
> return an error)

(https://discord.com/developers/docs/resources/webhook#execute-slackcompatible-webhook-query-string-params,
https://discord.com/developers/docs/resources/webhook#execute-githubcompatible-webhook-query-string-params).

Also documented on Execute Webhook:

> If the webhook channel is a forum or media channel, you must provide
> either `thread_id` in the query string params, or `thread_name` in the
> JSON/form params. If `thread_id` is provided, the message will send in
> that thread. If `thread_name` is provided, a thread with that name will
> be created in the channel.

(https://discord.com/developers/docs/resources/webhook#execute-webhook)

## Error responses, error codes, and rate limits

General HTTP response codes, verbatim (subset)
(https://discord.com/developers/docs/topics/opcodes-and-status-codes#http-http-response-codes):

| Code | Meaning |
|---|---|
| 200 (OK) | The request completed successfully. |
| 204 (NO CONTENT) | The request completed successfully but returned no content. |
| 400 (BAD REQUEST) | The request was improperly formatted, or the server couldn't understand it. |
| 401 (UNAUTHORIZED) | The `Authorization` header was missing or invalid. |
| 403 (FORBIDDEN) | The `Authorization` token you passed did not have permission to the resource. |
| 404 (NOT FOUND) | The resource at the location specified doesn't exist. |
| 429 (TOO MANY REQUESTS) | You are being rate limited, see Rate Limits. |

JSON error codes relevant to webhooks, verbatim
(https://discord.com/developers/docs/topics/opcodes-and-status-codes#json-json-error-codes):

| Code | Meaning |
|---|---|
| 10015 | Unknown webhook |
| 10016 | Unknown webhook service |
| 30007 | Maximum number of webhooks reached (15) |
| 30058 | Maximum number of webhooks per guild reached (1000) |
| 50006 | Cannot send an empty message |
| 50013 | You lack permissions to perform that action |
| 50027 | Invalid webhook token provided |
| 50035 | Invalid form body (returned for both `application/json` and `multipart/form-data` bodies), or invalid `Content-Type` provided |
| 220001 | Webhooks posted to forum channels must have a thread_name or thread_id |
| 220002 | Webhooks posted to forum channels cannot have both a thread_name and thread_id |
| 220003 | Webhooks can only create threads in forum channels |
| 220004 | Webhook services cannot be used in forum channels |

The general shape of a validation error body (v8+), verbatim example
(https://discord.com/developers/docs/reference#error-messages):

```json
{
  "code": 50035,
  "errors": {
    "access_token": {
      "_errors": [
        {
          "code": "BASE_TYPE_REQUIRED",
          "message": "This field is required"
        }
      ]
    }
  },
  "message": "Invalid Form Body"
}
```

Rate limits. Per-route limits key on top-level resources, and webhooks are
named explicitly as one such resource:

> Top-level resources are currently limited to channels (`channel_id`),
> guilds (`guild_id`), and webhooks (`webhook_id` or `webhook_id +
> webhook_token`).

(https://discord.com/developers/docs/topics/rate-limits)

Response headers on a normal request, verbatim
(https://discord.com/developers/docs/topics/rate-limits#header-format-rate-limit-header-examples):

```
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1470173023
X-RateLimit-Reset-After: 1
X-RateLimit-Bucket: abcd1234
```

On a 429, the JSON body structure is `message` (string), `retry_after`
(float, seconds to wait), `global` (boolean), `code?` (integer)
(https://discord.com/developers/docs/topics/rate-limits#exceeding-a-rate-limit-rate-limit-response-structure).
Example 429 response, verbatim:

```
< HTTP/1.1 429 TOO MANY REQUESTS
< Content-Type: application/json
< Retry-After: 65
< X-RateLimit-Limit: 10
< X-RateLimit-Remaining: 0
< X-RateLimit-Reset: 1470173023.123
< X-RateLimit-Reset-After: 64.57
< X-RateLimit-Bucket: abcd1234
< X-RateLimit-Scope: user
{
  "message": "You are being rate limited.",
  "retry_after": 64.57,
  "global": false
}
```

Global rate limit: "All bots can make up to 50 requests per second to our
API. If no authorization header is provided, then the limit is applied to
the IP address."
(https://discord.com/developers/docs/topics/rate-limits#global-rate-limit).
Webhook execute requests carry no `Authorization` header, so a high-volume
sender pointed at Nexul (or at real Discord) is rate-limited by IP against
this global ceiling in addition to whatever the per-webhook bucket allows.

Invalid-request ban: "IP addresses that make too many invalid HTTP requests
are automatically and temporarily restricted... Currently, this limit is
**10,000 per 10 minutes**. An invalid request is one that results in
**401**, **403**, or **429** statuses" and "If a webhook returns a **404**
status you should not attempt to use it again - repeated attempts to do so
will result in a temporary restriction."
(https://discord.com/developers/docs/topics/rate-limits#invalid-request-limit-aka-cloudflare-bans).

**Unverified:** the docs do not publish a specific numeric quota (e.g.
"N requests per M seconds") for the webhook-execute per-route bucket
itself — only that it exists, is keyed by `webhook_id`/`webhook_id+token`,
and is discoverable at runtime via the `X-RateLimit-*` headers. Any fixed
number quoted elsewhere on the web for "the webhook rate limit" is not
sourced from this documentation and should be treated as unverified.

## Multipart form shape for file uploads

> Endpoints that support file uploads use a `multipart/form-data` request
> body. The file field name varies by endpoint and is shown in its
> parameter table. Non-file parameters can be sent as individual form
> fields or collectively as JSON in the `payload_json` form field.
>
> Some endpoints use `files[n]` to upload files as attachments. All
> `files[n]` parameters must include a valid `Content-Disposition` subpart
> header with a `filename` and unique `name` parameter. Each file
> parameter must be uniquely named in the format `files[n]` such as
> `files[0]`, `files[1]`, or `files[42]`. The suffixed index `n` is the
> snowflake placeholder that can be used in the `attachments` field...
>
> Images can also be referenced in embeds using the `attachment://filename`
> URL.

(https://discord.com/developers/docs/reference#uploading-files)

Verbatim example request body with `payload_json` and files
(https://discord.com/developers/docs/reference#uploading-files):

```
--boundary
Content-Disposition: form-data; name="payload_json"
Content-Type: application/json

{
  "content": "Hello, World!",
  "embeds": [{
    "title": "Hello, Embed!",
    "description": "This is an embedded message.",
    "thumbnail": {
      "url": "attachment://myfilename.png"
    },
    "image": {
      "url": "attachment://mygif.gif"
    }
  }],
  "message_reference": {
    "message_id": "233648473390448641"
  },
  "attachments": [{
      "id": 0,
      "description": "Image of a cute little cat",
      "filename": "myfilename.png"
  }, {
      "id": 1,
      "description": "Rickroll gif",
      "filename": "mygif.gif"
  }]
}
--boundary
Content-Disposition: form-data; name="files[0]"; filename="myfilename.png"
Content-Type: image/png

[image bytes]
--boundary
Content-Disposition: form-data; name="files[1]"; filename="mygif.gif"
Content-Type: image/gif

[image bytes]
--boundary--
```

File size: "The file upload size limit applies to each file in a request.
The default limit is `20 MiB` for all users, but may be higher... depending
on their Nitro status or by the server's Boost Tier."
(https://discord.com/developers/docs/reference#uploading-files). Only
`.jpg`, `.jpeg`, `.png`, `.webp`, and `.gif` may be used as attachment
images referenced inside an embed via `attachment://`
(same URL, "Using Attachments within Embeds" section) — this restriction
applies to the embed-image use of an attachment, not to file uploads in
general.

The Attachment Request Structure used in the `attachments` field: `id`
(snowflake or number, "for new attachments this must match the `n` in
`files[n]`"), `filename?`, `title?`, `description?` (max 1024 characters),
`duration_secs?`, `waveform?`, `is_spoiler?`
(https://discord.com/developers/docs/resources/message#attachment-object-attachment-request-structure).

## Message endpoints on the same URL

- Get Webhook Message: `GET /webhooks/{webhook.id}/{webhook.token}/messages/{message.id}`
  — "Returns a previously-sent webhook message from the same token. Returns
  a message object on success." Query param `thread_id?` (snowflake)
  (https://discord.com/developers/docs/resources/webhook#get-webhook-message).
- Edit Webhook Message: `PATCH /webhooks/{webhook.id}/{webhook.token}/messages/{message.id}`
  — "Edits a previously-sent webhook message from the same token. Returns a
  message object on success." All body params are optional and nullable.
  Same JSON/form field set as Execute minus `username`/`avatar_url`/`tts`,
  plus the note: "Starting with API v10, the `attachments` array must
  contain all attachments that should be present after edit, including
  **retained and new** attachments provided in the request body."
  (https://discord.com/developers/docs/resources/webhook#edit-webhook-message).
- Delete Webhook Message: `DELETE /webhooks/{webhook.id}/{webhook.token}/messages/{message.id}`
  — "Deletes a message that was created by the webhook. Returns a
  `204 No Content` response on success."
  (https://discord.com/developers/docs/resources/webhook#delete-webhook-message).

All three take an optional `thread_id` query param when the message lives
in a thread (same three URLs above).

## Compatibility suffixes `/github` and `/slack`

`/slack`:

> Refer to Slack's documentation for more information. We do not support
> Slack's `channel`, `icon_emoji`, `mrkdwn`, or `mrkdwn_in` properties.

(https://discord.com/developers/docs/resources/webhook#execute-slackcompatible-webhook).
Query params: `thread_id?`, `wait` (default `true`) (same URL).

`/github`:

> Add a new webhook to your GitHub repo (in the repo's settings), and use
> this endpoint as the "Payload URL." You can choose what events your
> Discord channel receives by choosing the "Let me select individual
> events" option and selecting individual events for the new webhook
> you're configuring. The supported events are `commit_comment`, `create`,
> `delete`, `fork`, `issue_comment`, `issues`, `member`, `public`,
> `pull_request`, `pull_request_review`, `pull_request_review_comment`,
> `push`, `release`, `watch`, `check_run`, `check_suite`, `discussion`, and
> `discussion_comment`.

(https://discord.com/developers/docs/resources/webhook#execute-githubcompatible-webhook).
Query params: `thread_id?`, `wait` (default `true`) (same URL). Discord's
doc does not itself describe the translation logic (how a `push` payload
becomes a Discord message) beyond naming the supported event set — that
transformation is internal to Discord and is not published as a spec by
either Discord or GitHub. **Unverified**: the exact rendering rules
(what text/embed a `push` vs `release` vs `issues` event produces) are not
documented by a primary source; only the list of accepted GitHub event
names is.

## Three real payloads

### GitHub, through the `/github` suffix

GitHub does not publish a Discord-specific payload; the `/github`
compatible endpoint (per Discord's own doc, quoted above) is pointed at the
same "Payload URL" GitHub sends to any webhook, and Discord parses that
standard event JSON itself. GitHub's own webhook docs show the payload
shape delivered to any configured Payload URL, verbatim (values doc's own
example, truncated with `...` exactly as printed in the source):

```
POST /payload HTTP/1.1

X-GitHub-Delivery: 72d3162e-cc78-11e3-81ab-4c9367dc0958
X-Hub-Signature: sha1=7d38cdd689735b008b3c702edd92eea23791c5f6
X-Hub-Signature-256: sha256=d57c68ca6f92289e6987922ff26938930f6e66a2d161ef06abdf1859230aa23c
User-Agent: GitHub-Hookshot/044aadd
Content-Type: application/json
Content-Length: 6615
X-GitHub-Event: issues
X-GitHub-Hook-ID: 292430182
X-GitHub-Hook-Installation-Target-ID: 79929171
X-GitHub-Hook-Installation-Target-Type: repository

{
  "action": "opened",
  "issue": {
    "url": "https://api.github.com/repos/octocat/Hello-World/issues/1347",
    "number": 1347,
    ...
  },
  "repository" : {
    "id": 1296269,
    "full_name": "octocat/Hello-World",
    "owner": {
      "login": "octocat",
      "id": 1,
      ...
    },
    ...
  },
  "sender": {
    "login": "octocat",
    "id": 1,
    ...
  }
}
```

(https://docs.github.com/en/webhooks/webhook-events-and-payloads, "Example
webhook delivery" section). This is the literal body GitHub POSTs to a
Discord `/github` webhook URL for a repository configured with that Payload
URL; Discord then re-renders it into a message using the logic named (but
not spec'd) in its own docs.

### Grafana's Discord contact point

Grafana does not publish a filled-in example payload in its docs; the
payload shape below is the exact JSON produced by the current source of its
Discord notifier (struct field names and JSON tags, not paraphrased):

```go
type discordMessage struct {
	Username  string             `json:"username,omitempty"`
	Content   string             `json:"content"`
	AvatarURL string             `json:"avatar_url,omitempty"`
	Embeds    []discordLinkEmbed `json:"embeds,omitempty"`
}

type discordLinkEmbed struct {
	Title       string           `json:"title,omitempty"`
	Type        discordEmbedType `json:"type,omitempty"`
	Description string           `json:"description,omitempty"`
	URL         string           `json:"url,omitempty"`
	Color       int64            `json:"color,omitempty"`
	Footer      *discordFooter   `json:"footer,omitempty"`
	Image       *discordImage    `json:"image,omitempty"`
}
```

(https://github.com/grafana/alerting/blob/main/receivers/discord/v1/discord.go).
The resulting JSON on the wire for a firing alert looks like:

```json
{
  "username": "Grafana",
  "content": "",
  "embeds": [
    {
      "title": "[FIRING:1] HighErrorRate",
      "type": "rich",
      "url": "https://grafana.example.com/alerting/list",
      "color": 14037554,
      "footer": {
        "text": "Grafana v11.0.0",
        "icon_url": "https://grafana.com/static/assets/img/fav32.png"
      }
    }
  ]
}
```

Notable, straight from the source: `content` is always present (its JSON
tag has no `omitempty`) even when empty, because the username defaults to
`"Grafana"` unless overridden, the message content is truncated to 2000
runes client-side before sending (`discordMaxMessageLen = 2000`, matching
Discord's own limit) and the embed title is truncated to 256 runes
(`discordMaxTitleLen = 256`, again matching Discord's documented limit),
and when alert screenshots are attached, the request switches from a plain
JSON body to `multipart/form-data` with the JSON moved into a
`payload_json` form field and each image as an unnamed form file — the same
shape Discord's own multipart docs describe above
(https://github.com/grafana/alerting/blob/main/receivers/discord/v1/discord.go,
function `buildRequest`).

### Uptime Kuma's Discord notification

Verbatim (renamed only for readability) from the current source's DOWN
branch:

```js
let discorddowndata = {
    username: discordDisplayName,
    embeds: [
        {
            title: "❌ Your service " + monitorJSON["name"] + " went down. ❌",
            color: 16711680,
            timestamp: heartbeatJSON["time"],
            fields: [
                { name: "Service Name", value: monitorJSON["name"] },
                { name: "Service URL", value: addess },
                { name: "Went Offline", value: `<t:${wentOfflineTimestamp}:F>` },
                { name: `Time (${heartbeatJSON["timezone"]})`, value: heartbeatJSON["localDateTime"] },
                { name: "Error", value: heartbeatJSON["msg"] == null ? "N/A" : heartbeatJSON["msg"] }
            ]
        }
    ]
};
```

and the UP branch:

```js
let discordupdata = {
    username: discordDisplayName,
    embeds: [
        {
            title: "✅ Your service " + monitorJSON["name"] + " is up! ✅",
            color: 65280,
            timestamp: heartbeatJSON["time"],
            fields: [
                { name: "Service Name", value: monitorJSON["name"] },
                { name: "Service URL", value: addess },
                { name: "Went Offline", value: `<t:${wentOfflineTimestamp}:F>` },
                { name: "Downtime Duration", value: downtimeDuration },
                { name: `Time (${heartbeatJSON["timezone"]})`, value: heartbeatJSON["localDateTime"] },
                { name: "Ping", value: heartbeatJSON["ping"] + " ms" }
            ]
        }
    ]
};
```

(https://github.com/louislam/uptime-kuma/blob/master/server/notification-providers/discord.js).
"Service URL", "Went Offline", "Downtime Duration", and "Ping" fields are
each conditionally spliced into the `fields` array only when the relevant
data is available; they are shown unconditionally above for readability. If
the webhook itself has no avatar configured, Uptime Kuma also sets
`avatar_url` to its own icon, and if `discordSuppressNotifications` is on it
adds `flags: 1 << 12` (`SUPPRESS_NOTIFICATIONS`, matching the bit documented
at
https://discord.com/developers/docs/resources/message#message-object-message-flags).
Posting to a forum/media channel adds `thread_name`; posting to an existing
thread appends `?thread_id=` to the webhook URL itself rather than putting
it in the body, matching Discord's own query-string parameter for that
(https://discord.com/developers/docs/resources/webhook#execute-webhook-query-string-params).

## Questions not answerable from a primary source

- The webhook token's exact length and character-set contract (only one
  100-character hex-looking example exists in the docs; no stated spec).
- A fixed numeric per-webhook rate-limit quota (the mechanism — per-route
  bucket keyed on `webhook_id`/`webhook_id+token`, `X-RateLimit-*` headers,
  429 body shape — is documented; a specific "N per M seconds" number for
  the webhook-execute bucket is not).
- The exact HTTP status code returned by Execute Webhook when `wait=true`
  succeeds (the doc says a message body is returned but does not print the
  numeric status for this specific route; 200 is the reasonable read from
  the general status-code table, not a verbatim statement for this route).
- The internal rendering rules Discord applies inside `/github` and
  `/slack` to turn a GitHub or Slack payload into a Discord message (only
  the list of accepted GitHub event names, and the three unsupported Slack
  fields, are documented).
