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
- [x] add tracking support
hist file: .local/state/kino-hst
IMDBID, name, season, episode
- [ ] better structs
- [ ] check if a certain imdb ID can be actually fetched or not (does give errors or not)
- [ ] add unified error logging
- [ ] add download support
- [ ] allanime support
- [ ] subtitle support
- [ ] fix computers getting flagged
