# Alexandria

As in the library.

This is Techaro's log ingestion and aggregation pipeline, with a client for slog
to feed logs into the beast.

## Why does this exist?

In order to make sure that Anubis and related infrastructure bits are working
the best they can, we need to be able to monitor how Anubis is being used in the
real world. Alexandria is the pipeline that ingests logs from customer instances
of Anubis, internal infrastructure, and other sources so that there are fewer
panes of glass to look through when finding information.

Alexandria is the main ingestion component. When Anubis, Thoth, or other
services emit log lines, they buffer them in-memory for up to 32 kilobytes of
logs or one minute (whichever happens first). Then the logs are submitted to
Alexandria, and Alexandria will write them to
[Tigris](https://www.tigrisdata.com).

Once the logs are written to Tigris, Tigris emits
[object notification webhooks](https://www.tigrisdata.com/docs/buckets/object-notifications/)
to serverless functions. These serverless functions will fetch new logs, scrape
them for relevant information, publish those facts internally, and then handle
the next log observation.

## How are logs stored?

All logs are stored according to the following lifecycle rules:

- Logs are ingressed into the Standard storage tier into the `logs` folder.
- One day later, logs are compacted per log ID and moved to the Archive Instant
  Retrieval storage tier.
- After 91 days, logs are automatically deleted.

## How do I opt out of this?

For package maintainers, you may opt out of this by building Anubis with the
`limited-support-ability`
[build tag](https://www.digitalocean.com/community/tutorials/customizing-go-binaries-with-build-tags).
Please note that if you do this, Techaro's ability to support your installation
of Anubis or other software will be limited.

For users, you may opt out of this by running Anubis with the
`ANUBIS_LOG_SUBMISSION=i-want-to-make-it-harder-to-get-help` envrionment
variable set.
