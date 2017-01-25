# Bot for Prometheus' Alertmanager

This is the [Alertmanager](https://prometheus.io/docs/alerting/alertmanager/) bot for 
[Prometheus](https://prometheus.io/) that notifies you on alerts.  
Just send him a webhook and he will do the rest.

Additionally you can always **send one of the following commands** to get 
up-to-date information from the alertmanager.

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


## Development

### Build from source

`go get github.com/metalmatze/alertamanger-bot`

Build the binary using `make`:

```
make build
```
