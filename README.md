# tuiFeed

A terminal-based (TUI) feed reader for [Bluesky](https://bsky.app), built with [Bubble Tea v2](https://charm.land/bubbletea/v2).

Browse posts from followed accounts, view images inline, follow/unfollow accounts, save posts for later, and explore comment threads — all without authentication or leaving your terminal. Subscribe to any RSS/Atom/JSON feed (news, blogs, podcasts, Reddit, YouTube…), and follow Bluesky accounts through RSS bridges like [RSSHub](https://docs.rsshub.app/routes/social-media#bluesky).

## Features

- **Feed browser** — View recent posts from all followed accounts, sorted by date with pagination
- **Since-last-check** — See only new posts since your last visit, filtered by account
- **Image support** — Render images inline via [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/) or [Sixel](https://en.wikipedia.org/wiki/Sixel), or open externally in your default viewer (`open` on macOS, `xdg-open` on Linux, default file handler on Windows)
- **Manage follows** — Add or remove followed accounts through the UI
- **Saved posts** — Bookmark posts for later reading
- **Thread viewer** — Navigate comment/reply trees with breadcrumb trail and multi-level drilling
- **RSS reader** — Subscribe to any RSS 2.0/Atom/JSON feed URL (not just Bluesky), browse entries with inline images, save entries for later, and open links in your browser
- **RSS since-last-check** — See only new entries since your last visit, per feed
- **Keyboard-driven** — Full navigation with `j`/`k`, arrows, and single-key commands
- **No account required** — Uses the public, unauthenticated Bluesky API
- **Persistent config** — Follows, RSS subscriptions, last-check timestamps, and saved posts/entries stored in JSON at `$XDG_CONFIG_HOME/tuiFeed/follows.json`

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
5. **Manage RSS feeds** — Add any RSS/Atom/JSON feed URL to subscribe to (news sites, blogs, podcasts, Reddit, Hacker News, YouTube, or Bluesky accounts via a bridge — see below)
6. **View RSS feeds** — Browse entries from all subscribed feeds
7. **RSS since last check** — View new entries from a selected feed
8. **Saved RSS entries** — Browse your bookmarked RSS entries

### Subscribing to RSS feeds

The RSS reader accepts any RSS 2.0, Atom, or JSON-feed URL — not just Bluesky-related ones. Examples:

- `https://hnrss.org/frontpage` (Hacker News)
- `https://www.reddit.com/r/golang/.rss` (subreddit)
- `https://xkcd.com/rss.xml`
- Your favorite blog's `/feed` or `/rss.xml`

### Following X/Twitter accounts via Nitter

X has no free API or RSS, so for now tuiFeed reads X accounts through the [Nitter](https://github.com/zedeus/nitter/wiki/Instances) mirror [xcancel.com](https://xcancel.com). In *Manage RSS feeds* press `a` and type any of:

- `@handle` → full timeline
- `@handle/media` → media-only posts
- `@handle/with_replies` → timeline plus replies
- `x.com/handle` or `twitter.com/handle` also work

Image thumbnails render inline (`a`), and video clips play directly in the terminal with `v` via mpv's kitty output (install `mpv`; on other setups the clip opens externally).

> **xcancel.com whitelist:** first-time fetches return a "RSS reader not yet whitelisted" error containing a hex ID. Email `rss@xcancel.com` with that ID (and a one-line reason) to get your reader whitelisted; afterwards feeds work normally.

### Following Bluesky accounts via RSS

Bluesky has no native RSS. To follow a Bluesky account through the RSS reader, use an RSS bridge such as RSSHub:

```
https://rsshub.app/bluesky/user/<handle>
```

for example `https://rsshub.app/bluesky/user/torvalds.bsky.social`. Public RSSHub instances may be rate-limited or blocked; consider self-hosting RSSHub for reliable access.

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
| `w` / `Enter` | Open entry link in browser (RSS view) |
| `v` | Play video attachment in-terminal via mpv (RSS view) |
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
├── rss/              RSS/Atom feed fetching & parsing (gofeed)
├── rssfeed/          Scrollable RSS entries view
├── rssfeeds/         RSS feed subscription management view
├── utils/            Shared helpers (scroll, pagination, HTTP, external open)
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
