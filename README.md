# launchd-gen

> Convert cron expressions into macOS launchd plists.

`launchd` is how macOS schedules background work — and its XML property-list
format is a notorious pain to write by hand. `launchd-gen` takes the cron
syntax you already know, and emits a valid launchd plist ready to drop into
`~/Library/LaunchAgents/`.

```bash
$ launchd-gen --label com.me.daily "0 9 * * 1-5" /usr/bin/python3 script.py
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.me.daily</string>
    ...
```

## Install

### Homebrew (recommended)

```bash
brew tap VandyTheCoder/tools
brew install --cask launchd-gen
```

### Go

```bash
go install github.com/VandyTheCoder/launchd-gen@latest
```

### Pre-built binary

Grab the latest `tar.gz` for your arch from the
[releases page](https://github.com/VandyTheCoder/launchd-gen/releases) and drop
the binary into `/usr/local/bin/`.

## Why

launchd's `StartCalendarInterval` only accepts **single values per key** —
no ranges, no lists, no steps. A cron expression like `*/15 9-17 * * 1-5`
(every 15 minutes, business hours, weekdays) has to be expanded by hand into
180 separate dict entries. `launchd-gen` does that expansion automatically.

It also handles:
- Lists: `0 9,17 * * *`
- Ranges: `0 9 * * 1-5`
- Steps: `*/15 * * * *`
- Combinations: `0 9-17/2 * * 1-5`
- Shortcuts: `@reboot`, `@daily`, `@hourly`, `@weekly`, `@monthly`, `@yearly`

## Usage

```
launchd-gen [flags] <cron-expression> <command> [args...]
```

### Flags

| Flag          | Description                                               |
|---------------|-----------------------------------------------------------|
| `--label`     | **Required.** Reverse-DNS label (e.g. `com.me.daily`)     |
| `--workdir`   | `WorkingDirectory` for the job                            |
| `--stdout`    | Path to capture stdout                                    |
| `--stderr`    | Path to capture stderr                                    |
| `--env`       | Environment variable `KEY=VALUE` (repeatable)             |
| `--install`   | Write plist to `~/Library/LaunchAgents/<label>.plist`     |
| `--load`      | After `--install`, also `launchctl load` the plist        |

### Examples

**Print a plist to stdout:**

```bash
launchd-gen --label com.me.daily \
  "0 9 * * 1-5" \
  /usr/bin/python3 /Users/me/script.py
```

**Install and load in one shot:**

```bash
launchd-gen --install --load \
  --label com.me.news \
  --stdout /tmp/news.log \
  --stderr /tmp/news.err \
  "57 9 * * *" \
  /usr/local/bin/fetch-news
```

**Pass environment variables:**

```bash
launchd-gen --label com.me.db-backup \
  --env "PGUSER=postgres" \
  --env "PGDATABASE=mydb" \
  "0 3 * * *" \
  /usr/local/bin/pg_dumpall
```

For more, see [`examples/pulse-jobs.md`](examples/pulse-jobs.md) — the four
real scheduled agents that power the [Pulse dashboard](https://pulse.vandysodanheang.info).

## Limitations

- **User agents only.** `launchd-gen` writes to `~/Library/LaunchAgents/`. It
  does not support `/Library/LaunchDaemons/` system-level agents (which
  require root).
- **No day-of-week names.** Use numeric weekdays (`0-6`, or `7` for Sunday).
  `MON`, `TUE`, etc. are not yet supported.
- **No second-resolution cron.** Standard 5-field cron only.

## Development

```bash
git clone https://github.com/VandyTheCoder/launchd-gen
cd launchd-gen
go test ./...
go build .
./launchd-gen --help
```

## License

[MIT](LICENSE)
