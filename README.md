# tuiFeed

A terminal-based (TUI) feed reader for Bluesky.

## Features

- Browse posts from followed Bluesky accounts in the terminal
- View images directly in the terminal (Kitty graphics protocol or Sixel)
- Add/remove followed accounts via an interactive UI
- Keyboard-driven navigation (j/k, arrows)
- No authentication required — uses the public Bluesky API

## Requirements

- Go 1.25+
- A terminal with [Kitty graphics](https://sw.kovidgoyal.net/kitty/graphics-protocol/) or [Sixel](https://en.wikipedia.org/wiki/Sixel) support for in-terminal image rendering (optional; images can also be opened externally)

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

Configuration is stored at `$XDG_CONFIG_HOME/tuiFeed/follows.json`.

## License

MIT
