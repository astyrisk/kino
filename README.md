stream movies, TVs right from terminal

## Dependencies
- MPV
- Go 1.25+

## Installation
```bash
git clone https://github.com/astyrisk/kino.git
cd kino
go build
```

## TODOs

- [x] add TV support
- `GET https://api.tvmaze.com/lookup/shows?imdb={imdb_id}` — get tvmaze_id
- `GET https://api.tvmaze.com/shows/{tvmaze_id}/seasons` — for seasons
- `GET https://api.tvmaze.com/seasons/{season_id}/episodes` — for episodes
- [ ] add tracking support
- [ ] allanime support
- [ ] subtitle support
- [ ] fix computers getting flagged
