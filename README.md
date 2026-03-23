# Recap

A minimal CLI that automatically summarizes your Claude Code sessions into daily work logs.

## Install

```sh
go build -o recap .
```

## Setup

```sh
recap install
```

Registers lifecycle hooks in `~/.claude/settings.json` and creates the `~/.recap/` directory. To remove hooks (logs are preserved):

```sh
recap uninstall
```

## Usage

```sh
recap sync                          # parse JSONL sessions → daily markdown log
recap summarize                     # AI-summarize today's daily log via claude -p
recap run                           # sync + summarize (cron/launchd target)
recap run --date yesterday          # process a different day
recap run --date 2026-03-10         # specific date
```

The `--date` flag works on `sync`, `summarize`, and `run`.

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
