# tuiFeed

A TUI feed reader for Bluesky.

## Features

- Browse posts from followed Bluesky accounts in the terminal, either as posts since last check or as a group of a few last posts from every followed profile.
- View images directly in the terminal (using Kitty graphics protocol or Sixel)
- In case of your terminal not supporting in-terminal image display you may open those images externally
- Add/remove followed accounts via a UI
- Keyboard-driven navigation (j/k, arrows)
- No authentication or account required — uses the public Bluesky API
- Removes access to unfiltered feed which may capture your attention for unplanned, long amounts of scrolling

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

1. Select **Manage follows** to add Bluesky handles you want to follow
2. Select **View Feed** to browse posts from all followed accounts
3. Use `j`/`k` or `↑`/`↓` to navigate posts, `←`/`→` to cycle through images

Configuration is stored at `$XDG_CONFIG_HOME/tuiFeed/follows.json`, or at
`~/Library/Application Support/tuiFeed/follows.json` on macOS when
`XDG_CONFIG_HOME` is not set.

> **macOS Gatekeeper:** binaries downloaded from GitHub Releases are unsigned and
> will be quarantined by Gatekeeper. Either build from source, or clear the
> quarantine flag with `xattr -d com.apple.quarantine /path/to/tuiFeed`.

## License

MIT
