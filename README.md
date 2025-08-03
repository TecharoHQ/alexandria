# Alexandria

As in the library.

This is Techaro's log ingestion and aggregation pipeline, featuring a client for
slog to feed logs into the system.

## Why does this exist?

To ensure that Anubis and related infrastructure components operate at their
best, we need to monitor how Anubis is used in real-world scenarios. Alexandria
serves as the pipeline that ingests logs from customer instances of Anubis,
internal infrastructure, and other sources, reducing the number of tools
required to locate information.

Alexandria is the primary ingestion component. When Anubis, Thoth, or other
services emit log lines, they buffer them in-memory for up to 32 kilobytes of
logs or one minute (whichever occurs first). The logs are then submitted to
Alexandria, which writes them to [Tigris](https://www.tigrisdata.com).

Once the logs are stored in Tigris, Tigris emits
[object notification webhooks](https://www.tigrisdata.com/docs/buckets/object-notifications/)
to serverless functions. These functions fetch new logs, analyze them for
relevant information, publish those findings internally, and handle subsequent
log observations.

## How are logs stored?

Logs follow these lifecycle rules:

- Logs are ingested into the Standard storage tier within the `logs` folder.
- After one day, logs are compacted by log ID and moved to the Archive Instant
  Retrieval storage tier.
- Logs are automatically deleted after 91 days.

## How do I opt out of this?

For package maintainers, you can opt out by building Anubis with the
`limitedsupportability`
[build tag](https://www.digitalocean.com/community/tutorials/customizing-go-binaries-with-build-tags).
Please note that opting out will limit Techaro's ability to support your
installation of Anubis or other software.

For users, you can opt out by running Anubis with the
`ANUBIS_LOG_SUBMISSION=i-want-to-make-it-harder-to-get-help` environment
variable set.
