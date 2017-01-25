# Bot for Prometheus' Alertmanager [![Build Status](https://drone.github.matthiasloibl.com/api/badges/metalmatze/alertmanager-bot/status.svg)](https://drone.github.matthiasloibl.com/metalmatze/alertmanager-bot)

[![Docker Pulls](https://img.shields.io/docker/pulls/metalmatze/alertmanager-bot.svg?maxAge=604800)](https://hub.docker.com/r/metalmatze/alertmanager-bot)
[![Go Report Card](https://goreportcard.com/badge/github.com/metalmatze/alertmanager-bot)](https://goreportcard.com/report/github.com/metalmatze/alertmanager-bot)


This is the [Alertmanager](https://prometheus.io/docs/alerting/alertmanager/) bot for 
[Prometheus](https://prometheus.io/) that notifies you on alerts.  
Just send him a webhook and he will do the rest.

Additionally you can always **send commands** to get up-to-date information from the alertmanager.

## Commands

###### /start

> Hey, Matthias! I will now keep you up to date!  
> [/help](#help)

###### /stop

> Alright, Matthias! I won't talk to you again.  
> [/help](#help)

###### /alerts

> 🔥 **FIRING** 🔥  
> **NodeDown** (Node scraper.krautreporter.rancher.internal:8080 down)  
> scraper.krautreporter.rancher.internal:8080 has been down for more than 1 minute.  
> **Started**: 1 week 2 days 3 hours 50 minutes 42 seconds ago  
> 
> 🔥 **FIRING** 🔥  
> **RancherServiceState** (scraper is inactive)  
> scraper is inactive for more than 1 minute.  
> **Started**: 1 week 2 days 3 hours 50 minutes 57 seconds ago  


###### /silences

> NodeDown 🔕  
>  `job="ranch-eye" monitor="exporter-metrics" severity="page"`  
> **Started**: 1 month 1 week 5 days 8 hours 27 minutes 57 seconds ago  
> **Ends**: -11 months 2 weeks 2 days 19 hours 15 minutes 24 seconds  
> 
> RancherServiceState 🔕  
>  `job="rancher" monitor="exporter-metrics" name="scraper" rancherURL="http://rancher.example.com/v1" severity="page" state="inactive"`  
> **Started**: 1 week 2 days 3 hours 46 minutes 21 seconds ago  
> **Ends**: -3 weeks 1 day 13 minutes 24 seconds  

###### /users

> Currently 1 users are subscribed.


###### /status

> **AlertManager**  
> Version: 0.5.1  
> Uptime: 3 weeks 1 day 6 hours 15 minutes 2 seconds  
> **AlertManager Bot**  
> Version: 0.2  
> Uptime: 3 weeks 1 hour 17 minutes 19 seconds  

###### /help

> I'm the AlertManager bot for Prometheus. I will notify you about alerts.
> You can also ask me about my [/status](#status), [/alerts](#alerts) & [/silences](#silences)
> 
> Available commands:
> [/start](#start) - Subscribe for alerts.
> [/stop](#stop) - Unsubscribe for alerts.
> [/status](#status) - Print the current status.
> [/alerts](#alerts) - List all alerts.
> [/silences](#silences) - List all silences.

## Installation

### Docker

`docker pull metalmatze/alertamanger-bot`

Start as a command:

	docker run -d \
		-e 'TELEGRAM_TOKEN=XXX' \
		-e 'TELEGRAM_ADMIN=1234567' \
		-e 'ALERTMANAGER_URL=http://alertmanager:9093' \
		-e 'STORE=/data/users.yml' \
		-v '/srv/monitoring/alertmanager-bot:/data'
		--name alertmanager-bot \
		alertmanager-bot

Usage within docker-compose:

	alertmanager-bot:
	  image: metalmatze/alertmanager-bot
	  environment:
	    TELEGRAM_TOKEN: XXX
	    TELEGRAM_ADMIN: '1234567'
	    ALERTMANAGER_URL: http://alertmanager:9093
	    STORE: /data/users.yml
	  volumes:
	  - /srv/monitoring/alertmanager-bot:/data

### Build from source

`go get github.com/metalmatze/alertamanger-bot`

### Configuration

ENV Variable | Description
|-------------------|------------------------------------------------------|
| ALERTMANAGER_URL  | Address of the alertmanager, default: `http://localhost:9093` |
| LISTEN_ADDR       | Address that the bot listens for webhooks, default: `0.0.0.0:8080` |
| TELEGRAM_TOKEN    | Token you get from [@botfather](https://telegram.me/botfather) |
| TELEGRAM_ADMIN    | The Telegram user id for the admin |
| STORE             | The subscribed users are persisted to the file, default: `data.yml` |

## Development

Build the binary using `make`:

```
make install
```
<!-- TODO: Write more -->

## Missing

Commands:

* `/silence` - show a specific silence
* `/silence_del` - delete a silence by command
* `/silence_add` - add a silence for a alert by command

Authentication:

Right now only one user can use the bot by giving the bot one telegram user id.  
Also if the user subscribes the user is marshalled to the store `.yml` file.  
_Maybe™_ that should be improved for better deployment within orchestration tools.

Other Messengers:

At the moment I only implemented Telegram, because it's so freakin' easy to do.  
But I know people that would be interested in a [Slack](https://slack.com/) bot for the alertmanager as well.
Personally I would also like to take a look at building a [[matrix]](https://matrix.org/) integration.
