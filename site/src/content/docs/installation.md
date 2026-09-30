---
title: "installation"
description: "Get gaur running on your Arch system"
---

gaur runs on Arch Linux and its derivatives: Manjaro, EndeavourOS, CachyOS, you name it.  
Here's how to get it.

## what you'll need

Make sure these are on your system before installing:

- **Arch Linux** (or any Arch-based distro)
- **An AUR helper:** [paru](https://github.com/Morganamilo/paru) or [yay](https://github.com/Jguer/yay)
- **fzf:** [fzf](https://github.com/junegunn/fzf) powers the fuzzy search
- **paccache:** Part of `pacman-contrib`, used for cache management
- **Go 1.21+** (only needed if building from source)

Quick install of dependencies:

```bash
sudo pacman -S fzf pacman-contrib
```

## install methods

### AUR (recommended)

The fastest route. Pick your helper:

```bash
paru -S gaur-bin
# or
yay -S gaur-bin
```

### go install

Already have Go set up? Grab the latest release directly:

```bash
go install github.com/prbhtkumr/gaur@latest
```

Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your `$PATH`.

### from source

Want to run the bleeding edge?

```bash
git clone https://github.com/prbhtkumr/gaur.git
cd gaur
go build -o gaur .
sudo mv gaur /usr/local/bin/
```

## verify

Check that everything's working:

```bash
gaur --list-themes
```

If you see a list of themes, you're all set. Run `gaur` to launch.

## first run

On first launch, gaur will:

1. Create a config file at `~/.config/gaur/config.toml`
2. Open in Install mode (or your configured default)
3. Load your local package database

Head over to [configuration](/docs/configuration) to customize your setup, or jump straight into [usage](/docs/usage) to learn the shortcuts.
