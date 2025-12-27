# Kino

 *Film in Terminal*

A simple CLI tool to search, stream and track films/TV shows.

![Demo](assets/example.gif)

## Dependencies

- **mpv** - Media player
- **Go 1.25+** (for building from source)

## Installation

```bash
git clone https://github.com/astyrisk/kino.git
cd kino
go build -o kino .
```

## Usage


```bash
./kino
```

or 

```bash
./kino "The Matrix"
```
## Roadmap

### Completed
- [x] TV show streaming with season/episode navigation
- [x] Command-line search arguments
- [x] MPV cache optimization
- [x] Video title display

### Todo
- [x] Refactor [`main.go`](main.go:1)
- [ ] Fix extractor decoder issue
- [ ] Add download functionality
- [ ] Add subtitle support
- [ ] Implement watch tracking
- [ ] Prepare first release
- [ ] Add anime support (AllAnime)


https://zealotsofzenith.site/content/8a0800ba6c075107ec5922ebc951f4cc/7e39f90815d0cbca3fcb1178fb9fa803/page-13.html

## Contributing

Found a bug or have a feature request? Open an issue or submit a pull request.

## License

GPLv3 License - see LICENSE file for details.

**Note**: This is an educational project. Ensure you comply with local laws regarding streaming content. Some code may have been assisted by deepseek-v3.1-nex-n1.

---
