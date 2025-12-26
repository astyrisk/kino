# Kino

> Watch movies and TV shows from your terminal

A simple CLI tool to search and stream movies and TV shows using mpv player.

![Demo](media/example.gif)

## Dependencies

- **mpv** - Media player
- **Go 1.25+** (for building from source)

### Install mpv

| Platform | Command |
|----------|---------|
| Ubuntu/Debian | `sudo apt install mpv` |
| macOS | `brew install mpv` |
| Windows | `choco install mpv` |
| Arch Linux | `sudo pacman -S mpv` |

## Installation

```bash
git clone https://github.com/astyrisk/kino.git
cd kino
go build -o kino .
```

## Usage

### Interactive Mode

```bash
./kino
```

### Search from Command Line

```bash
./kino "The Matrix"
```

Shows top 5 results with details about the first match.

## Features

- 🔍 Fuzzy search for movies and TV shows
- 📺 Browse seasons and episodes
- 🎬 Multiple quality options
- ⚡ Fast and lightweight
- 🖥️ Clean terminal interface

## Troubleshooting

### mpv not found
Install mpv using the commands above.

### Build errors
Ensure you have Go 1.25 or later:
```bash
go version
```

Update dependencies:
```bash
go mod tidy
```

## Debug Mode

Enable debug logging:
```bash
DEBUG=1 ./kino
```

## Contributing

Found a bug or have a feature request? Open an issue or submit a pull request.

## License

GPLv3 License - see LICENSE file for details.

**Note**: This is an educational project. Ensure you comply with local laws regarding streaming content.

---
