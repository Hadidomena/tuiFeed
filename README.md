# tuiFeed

A terminal-based (TUI) feed reader for [Bluesky](https://bsky.app), built with [Bubble Tea v2](https://charm.land/bubbletea/v2).

Browse posts from followed accounts, view images inline, follow/unfollow accounts, save posts for later, and explore comment threads — all without authentication or leaving your terminal.

## Features

- **Feed browser** — View recent posts from all followed accounts, sorted by date with pagination
- **Since-last-check** — See only new posts since your last visit, filtered by account
- **Image support** — Render images inline via [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/) or [Sixel](https://en.wikipedia.org/wiki/Sixel), or open externally in your default viewer (`open` on macOS, `xdg-open` on Linux, default file handler on Windows)
- **Manage follows** — Add or remove followed accounts through the UI
- **Saved posts** — Bookmark posts for later reading
- **Thread viewer** — Navigate comment/reply trees with breadcrumb trail and multi-level drilling
- **Keyboard-driven** — Full navigation with `j`/`k`, arrows, and single-key commands
- **No account required** — Uses the public, unauthenticated Bluesky API
- **Persistent config** — Follows, last-check timestamps, and saved posts stored in JSON at `$XDG_CONFIG_HOME/tuiFeed/follows.json`

## Requirements

- Go 1.25+

## Platform support

- **Linux**, **macOS**, and **Windows** are supported.
- On macOS and Linux, images that can't be rendered in the terminal are opened
  in your default image viewer (`open` / `xdg-open`). On Windows they are opened
  via the default file handler (`rundll32`).

## Optional Requirements
- A terminal with [Kitty graphics](https://sw.kovidgoyal.net/kitty/graphics-protocol/) or [Sixel](https://en.wikipedia.org/wiki/Sixel) support for in-terminal image rendering.
  - Recommended: kitty, WezTerm, Ghostty (Kitty protocol), iTerm2 3.5+ (Sixel).
  - Apple's **Terminal.app** supports neither protocol, so images there are opened externally in Preview.

## Install

```bash
go install github.com/Hadidomena/tuiFeed@latest
```

## Build from source

```bash
git clone https://github.com/Hadidomena/tuiFeed.git
cd tuiFeed
go build .
./tuiFeed
```

## Usage

1. **Manage follows** — Add Bluesky handles to follow
2. **View Feed** — Browse recent posts from all followed accounts
3. **Posts since last check** — View new posts from a selected account
4. **Saved posts** — Browse your bookmarked posts

Configuration is stored at `$XDG_CONFIG_HOME/tuiFeed/follows.json`, or at
`~/Library/Application Support/tuiFeed/follows.json` on macOS when
`XDG_CONFIG_HOME` is not set.

> **macOS Gatekeeper:** binaries downloaded from GitHub Releases are unsigned and
> will be quarantined by Gatekeeper. Either build from source, or clear the
> quarantine flag with `xattr -d com.apple.quarantine /path/to/tuiFeed`.

### Keybindings

| Key | Action |
|---|---|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `a` | Show attachments inline |
| `o` | Open attachment externally |
| `s` | Save/unsave post |
| `c` | Open comments (thread view) |
| `r` | Refresh feed |
| `←` / `→` | Cycle through attachments |
| `Enter` | Drill into reply (thread view) |
| `h` / `Backspace` | Go back a level (thread view) |
| `Esc` | Return to previous view |
| `q` / `Ctrl+C` | Quit |

## Project Structure

```
├── main.go           Entry point, dashboard, view routing
├── bsk/              Bluesky API client & image rendering
├── config/           JSON config persistence
├── feed/             Scrollable post feed view
├── follows/          Follow management view
├── thread/           Comment thread tree view
├── attach/           Attachment download & rendering bridge
├── utils/            Shared helpers (scroll, pagination, HTTP)
└── internal/testutil Test utilities
```

## Testing

```bash
go test ./...                        # All tests
go test -v -race ./bsk               # Single package with race detection
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
```

## License

MIT
