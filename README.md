# Introduction

*Songbot* is a Discord Bot that queries Youtube and Spotify to find requested songs.

# Installation

- Clone this project
```
git clone https://github.com/vincent-heng/discord-songbot
```

- Set the Spotify, Youtube and Discord API keys in the configuration file
```
cp config-sample.json config.json
vi config.json
```

- Run it with Docker.
```
docker build . -t songbot:latest
docker run songbot:latest
```
