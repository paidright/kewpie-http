## Kewpie HTTP

> Put your Kewpie in the Cloud

![Kewpie HTTP Logo](https://notbad.software/img/kewpie_http.jpg "Kewpie HTTP Logo")

[Kewpie](https://github.com/davidbanham/kewpie_go) is a task queue abstraction over multiple backends. This is an HTTP server that wraps that abstraction for times when you don't want to call the library directly.

It's a great choice for publishing tasks to queues.

It can also be used for pulling tasks from queues, subscribing, but it comes with a caveat. Any task you pull from a queue via Kewpie HTTP will be immediately acked. That means that if your worker process crashes or the task fails, it's up to you to get it back on the queue to be retried.

### Running it

The queues you would like available must be defined up front. These are passed via environment variables, ie:

```
export KEWPIE_QUEUE_ONE=my_cool_queue
export KEWPIE_QUEUE_TWO=tasks_r_fun
```

You must also specify a port and a backend ie:
```
export PORT=8080
export KEWPIE_BACKEND=postgres
```

You will also need to make available any env vars required by your chosen backend eg:

```
export DB_URI=postgres://kewpie:wut@localhost:5432/kewpie?sslmode=disable
```

### Using it

You can `POST` a task payload to `/queues/QUEUE_NAME/publish`.

You can `GET` a task from `/queues/QUEUE_NAME/subscribe`.

You can purge a queue with a `POST` to `/queues/QUEUE_NAME/purge`.

If your backend supports it, you can purge only matching messages with a `POST` to `/queues/QUEUE_NAME/purge?matching=foo`

Either plain 'ol JSON or JSON-API payload formats are supported.

Plain:

```
{
  id: "uuid",
  body: "A string. Encode it however you like. Often JSON is handy.",
  run_at: "The time, in RFC3339 format, to wait until before handing this task to a subscriber",
  delay: "The number of seconds to wait before handing this task to a subscriber. If set, this overrides Run At",
  no_exp_backoff: "If true, don't exponentially back off attempts on failure. Just go your hardest.",
  attempts: "Ignored on publish. On subscribe, the amount of times a task has been attempted."
}
```

JSON-API:

```
{
  errors: [{detail: "A message about the error, if any"}],
  data: {
    id: "uuid",
    body: "A string. Encode it however you like. Often JSON is handy.",
    run_at: "The time, in RFC3339 format, to wait until before handing this task to a subscriber",
    delay: "The number of seconds to wait before handing this task to a subscriber. If set, this overrides Run At",
    no_exp_backoff: "If true, don't exponentially back off attempts on failure. Just go your hardest.",
    attempts: "Ignored on publish. On subscribe, the amount of times a task has been attempted.",
  },
  meta: {}
}
```
