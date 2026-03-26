# Recap

A minimal CLI that automatically summarizes your Claude Code sessions into daily work logs.

## Install

```sh
go build -o recap .
```

## Usage

Recap works without any setup, but I recommend running `recap install` first — see [Hooks](#hooks-optional) below. Hooks allow us to attach branch/ticket context, improving summaries.

Point it at a day and it parses your Claude Code JSONL transcripts into a markdown log:

```sh
recap sync                          # parse today's sessions → daily markdown log
recap sync --date 2026-03-10        # specific date
recap summarize                     # AI-summarize today's daily log via claude -p
recap run                           # sync + summarize (cron/launchd target)
recap run --date yesterday          # process a different day
```

The `--date` flag works on `sync`, `summarize`, and `run`.

## Hooks (optional)

```sh
recap install
```

Without hooks, recap only captures sessions at sync time — you get the full transcript but nothing in real time. Installing hooks registers `SessionStart` and `Stop` listeners in `~/.claude/settings.json` that write to the daily log as you work:

- **SessionStart** logs a session header with timestamp, git branch, and working directory the moment a session begins.
- **Stop** captures Claude's final message immediately when a session ends, before sync runs.

This means your daily log stays current throughout the day rather than only updating on the next sync. The sync command automatically deduplicates against hook entries, so there's no double-logging.

To remove hooks (logs are preserved):

```sh
recap uninstall
```

## Scheduling

Recap does not install a schedule automatically. Set one up yourself to run `recap run --date yesterday` once per day.

### macOS (launchd)

Save as `~/Library/LaunchAgents/com.recap.daily.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.recap.daily</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/recap</string>
        <string>run</string>
        <string>--date</string>
        <string>yesterday</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>8</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
</dict>
</plist>
```

Then load it:

```sh
launchctl load ~/Library/LaunchAgents/com.recap.daily.plist
```

### Linux (cron)

```sh
crontab -e
```

```
0 8 * * * /usr/local/bin/recap run --date yesterday
```

## Data Directory

```
~/.recap/
├── logs/
│   └── daily/
│       └── 2026-03-12.md       # daily conversation log
└── state.json                   # byte offsets for incremental parsing
```

## Requirements

- Go 1.25.5+
- [Claude Code](https://claude.ai/code)
- `claude` CLI (for the `summarize` command)
