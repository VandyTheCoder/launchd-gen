# Example: Pulse Daily Cron Jobs

These four jobs are the real scheduled agents that power the Pulse dashboard
at <https://pulse.vandysodanheang.info>. They double as acceptance fixtures
for `launchd-gen` — the generated plists should match the hand-written ones
in `~/Library/LaunchAgents/com.pulse.cron-*.plist`.

## cron-news — Daily news fetcher

Runs every day at 09:57 AM. Fetches 5 topics × 2 articles, POSTs to Pulse.

```bash
launchd-gen \
  --label com.pulse.cron-news \
  --stdout /tmp/pulse-cron-news.log \
  --stderr /tmp/pulse-cron-news.err \
  --workdir /Users/macbookpro/Playground/AI/Home \
  "57 9 * * *" \
  /Users/macbookpro/Playground/AI/Home/schedule/cron-news
```

## cron-briefing — Weekday AI briefing

Runs Monday–Friday at 10:03 AM. Pulls Google Calendar + Jira, generates the
daily briefing, POSTs to Pulse.

```bash
launchd-gen \
  --label com.pulse.cron-briefing \
  --stdout /tmp/pulse-cron-briefing.log \
  --stderr /tmp/pulse-cron-briefing.err \
  --workdir /Users/macbookpro/Playground/AI/Home \
  "3 10 * * 1-5" \
  /Users/macbookpro/Playground/AI/Home/schedule/cron-briefing
```

The `1-5` weekday range expands into **five** `StartCalendarInterval` dicts,
one per weekday — launchd's native format cannot represent a range in a
single dict, so `launchd-gen` does the expansion for you.

## cron-activity — Daily activity log summariser

Runs every day at 10:05 AM. Reads yesterday's `activity-logs/*.md`, summarises
with Claude, POSTs to Pulse `/webhooks/activity`.

```bash
launchd-gen \
  --label com.pulse.cron-activity \
  --stdout /tmp/pulse-cron-activity.log \
  --stderr /tmp/pulse-cron-activity.err \
  --workdir /Users/macbookpro/Playground/AI/Home \
  "5 10 * * *" \
  /Users/macbookpro/Playground/AI/Home/schedule/cron-activity
```

## cron-optimize — Weekly trade parameter sweep

Runs every Friday at 12:01 PM. Grid-searches TP/SL values for XRP and Gold,
deploys the best config, sends a Telegram report.

```bash
launchd-gen \
  --label com.pulse.cron-optimize \
  --stdout /tmp/pulse-cron-optimize.log \
  --stderr /tmp/pulse-cron-optimize.err \
  --workdir /Users/macbookpro/Playground/AI/Home \
  "1 12 * * 5" \
  /Users/macbookpro/Playground/AI/Home/schedule/cron-optimize
```

## Install all four in one shot

Add `--install --load` to each of the above commands to write the plist to
`~/Library/LaunchAgents/` and immediately register it with `launchctl`:

```bash
launchd-gen --install --load \
  --label com.pulse.cron-news \
  --stdout /tmp/pulse-cron-news.log \
  --stderr /tmp/pulse-cron-news.err \
  "57 9 * * *" \
  /Users/macbookpro/Playground/AI/Home/schedule/cron-news
```
