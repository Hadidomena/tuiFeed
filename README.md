# tuiFeed

A TUI feed reader for Bluesky.

## Features

- Browse posts from followed Bluesky accounts in the terminal, either as posts since last check or as a group of a few last post from every followed profile.
- View images directly in the terminal (using Kitty graphics protocol or Sixel)
- In case of terminal not supporting image display you may open those images externally
- Add/remove followed accounts via an UI
- Keyboard-driven navigation (j/k, arrows)
- No authentication or account required — uses the public Bluesky API
- Removes 

## Requirements

- Go 1.25+

## Optional Requirements
- A terminal with [Kitty graphics](https://sw.kovidgoyal.net/kitty/graphics-protocol/) or [Sixel](https://en.wikipedia.org/wiki/Sixel) support for in-terminal image rendering.

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
