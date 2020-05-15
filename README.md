# Bot for Prometheus' Alertmanager [![Build Status](https://cloud.drone.io/api/badges/metalmatze/alertmanager-bot/status.svg)](https://cloud.drone.io/metalmatze/alertmanager-bot)


[![Docker Pulls](https://img.shields.io/docker/pulls/metalmatze/alertmanager-bot.svg?maxAge=604800)](https://hub.docker.com/r/metalmatze/alertmanager-bot)
[![Go Report Card](https://goreportcard.com/badge/github.com/metalmatze/alertmanager-bot)](https://goreportcard.com/report/github.com/metalmatze/alertmanager-bot)


This is the [Alertmanager](https://prometheus.io/docs/alerting/alertmanager/) bot for
[Prometheus](https://prometheus.io/) that notifies you on alerts.  
Just configure the Alertmanager to send Webhooks to the bot and that's it.

Additionally you can always **send commands** to get up-to-date information from the alertmanager.

### Why?

Alertmanager already integrates a lot of different messengers as receivers for alerts.  
I want to extend this basic functionality.

Previously the Alertmanager could only talk to you via a chat, but now you can talk back via [commands](#commands).  
You can ask about current ongoing [alerts](#alerts) and [silences](#silences).  
In the future I plan to also support silencing via the chat, so you can silences after getting an alert from within the chat.  
A lot of other things can be added!

## Messengers

Right now it supports [Telegram](https://telegram.org/), but I'd like to [add more](#more-messengers) in the future.

## Commands

###### /start

> Hey, Matthias! I will now keep you up to date!  
> [/help](#help)

###### /stop

> Alright, Matthias! I won't talk to you again.  
> [/help](#help)

###### /alerts

> 🔥 **FIRING** 🔥  
> **NodeDown** (Node scraper.krautreporter:8080 down)  
> scraper.krautreporter:8080 has been down for more than 1 minute.  
> **Started**: 1 week 2 days 3 hours 50 minutes 42 seconds ago  
>
> 🔥 **FIRING** 🔥
> **monitored_service_down** (MONITORED SERVICE DOWN)
> The monitoring service 'digitalocean-exporter' is down.
> **Started**: 10 seconds ago

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

###### /chats

> Currently these chat have subscribed:
> @MetalMatze


###### /status

> **AlertManager**  
> Version: 0.5.1  
> Uptime: 3 weeks 1 day 6 hours 15 minutes 2 seconds  
> **AlertManager Bot**  
> Version: 0.4.2  
> Uptime: 3 weeks 1 hour 17 minutes 19 seconds  

###### /help

> I'm a Prometheus AlertManager Bot for Telegram. I will notify you about alerts.  
> You can also ask me about my [/status](#status), [/alerts](#alerts) & [/silences](#silences)  
>   
> Available commands:  
> [/start](#start) - Subscribe for alerts.  
> [/stop](#stop) - Unsubscribe for alerts.  
> [/status](#status) - Print the current status.  
> [/alerts](#alerts) - List all alerts.  
> [/silences](#silences) - List all silences.  
> [/chats](#chats) - List all users and group chats that subscribed.

## Installation

### Docker

`docker pull metalmatze/alertmanager-bot:0.4.2`

Start as a command:

#### Bolt Storage

```bash
docker run -d \
	-e 'ALERTMANAGER_URL=http://alertmanager:9093' \
	-e 'BOLT_PATH=/data/bot.db' \
	-e 'STORE=bolt' \
	-e 'TELEGRAM_ADMIN=1234567' \
	-e 'TELEGRAM_TOKEN=XXX' \
	-v '/srv/monitoring/alertmanager-bot:/data' \
	--name alertmanager-bot \
	metalmatze/alertmanager-bot:0.4.2
```

#### Consul Storage

```bash
docker run -d \
	-e 'ALERTMANAGER_URL=http://alertmanager:9093' \
	-e 'CONSUL_URL=localhost:8500' \
	-e 'STORE=consul' \
	-e 'TELEGRAM_ADMIN=1234567' \
	-e 'TELEGRAM_TOKEN=XXX' \
	--name alertmanager-bot \
	metalmatze/alertmanager-bot:0.4.2
```

### docker-compose:

[embedmd]:# (deployments/examples/docker-compose.yaml)
```yaml
networks:
  alertmanager-bot: {}
services:
  alertmanager-bot:
    command:
    - --alertmanager.url=http://localhost:9093
    - --log.level=info
    - --store=bolt
    - --bolt.path=/data/bot.db
    environment:
      TELEGRAM_ADMIN: "1234"
      TELEGRAM_TOKEN: XXXXXXX
    image: metalmatze/alertmanager-bot:0.4.2
    networks:
    - alertmanager-bot
    ports:
    - 8080:8080
    restart: always
    volumes:
    - ./data:/data
version: "3"
```

### Kubernetes:

[embedmd]:# (deployments/examples/kubernetes.yaml)
```yaml
apiVersion: v1
items:
- apiVersion: v1
  data:
    admin: MTIzNA==
    token: WFhYWFhYWA==
  kind: Secret
  metadata:
    labels:
      app.kubernetes.io/name: alertmanager-bot
    name: alertmanager-bot
    namespace: monitoring
  type: Opaque
- apiVersion: v1
  kind: Service
  metadata:
    labels:
      app.kubernetes.io/name: alertmanager-bot
    name: alertmanager-bot
    namespace: monitoring
  spec:
    ports:
    - name: http
      port: 8080
      targetPort: 8080
    selector:
      app.kubernetes.io/name: alertmanager-bot
- apiVersion: apps/v1
  kind: StatefulSet
  metadata:
    labels:
      app.kubernetes.io/name: alertmanager-bot
    name: alertmanager-bot
    namespace: monitoring
  spec:
    podManagementPolicy: OrderedReady
    replicas: 1
    selector:
      matchLabels:
        app.kubernetes.io/name: alertmanager-bot
    serviceName: alertmanager-bot
    template:
      metadata:
        labels:
          app.kubernetes.io/name: alertmanager-bot
        name: alertmanager-bot
        namespace: monitoring
      spec:
        containers:
        - args:
          - --alertmanager.url=http://localhost:9093
          - --log.level=info
          - --store=bolt
          - --bolt.path=/data/bot.db
          env:
          - name: TELEGRAM_ADMIN
            valueFrom:
              secretKeyRef:
                key: admin
                name: alertmanager-bot
          - name: TELEGRAM_TOKEN
            valueFrom:
              secretKeyRef:
                key: token
                name: alertmanager-bot
          image: metalmatze/alertmanager-bot:0.4.2
          imagePullPolicy: IfNotPresent
          name: alertmanager-bot
          ports:
          - containerPort: 8080
            name: http
          resources:
            limits:
              cpu: 100m
              memory: 128Mi
            requests:
              cpu: 25m
              memory: 64Mi
          volumeMounts:
          - mountPath: /data
            name: data
        restartPolicy: Always
        volumes:
        - name: data
          persistentVolumeClaim:
            claimName: data
    volumeClaimTemplates:
    - apiVersion: v1
      kind: PersistentVolumeClaim
      metadata:
        labels:
          app.kubernetes.io/name: alertmanager-bot
        name: alertmanager-bot
        namespace: monitoring
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
        storageClassName: standard
kind: List
```

### Ansible

If you prefer using configuration management systems (like Ansible) you might be interested in the following role:  [mbaran0v.alertmanager-bot](https://github.com/mbaran0v/ansible-role-alertmanager-bot)

### Build from source

`GO111MODULE=on go get github.com/metalmatze/alertmanager-bot/cmd/alertmanager-bot`

### Configuration

ENV Variable | Description
|-------------------|------------------------------------------------------|
| ALERTMANAGER_URL  | Address of the alertmanager, default: `http://localhost:9093` |
| BOLT_PATH         | Path on disk to the file where the boltdb is stored, default: `/tmp/bot.db` |
| CONSUL_URL        | The URL to use to connect with Consul, default: `localhost:8500` |
| LISTEN_ADDR       | Address that the bot listens for webhooks, default: `0.0.0.0:8080` |
| STORE             | The type of the store to use, choose from bolt (local) or consul (distributed) |
| TELEGRAM_ADMIN    | The Telegram user id for the admin. The bot will only reply to messages sent from an admin. All other messages are dropped and logged on the bot's console.<br> Your user id you can get from [@userinfobot](https://t.me/userinfobot). |
| TELEGRAM_TOKEN    | Token you get from [@botfather](https://telegram.me/botfather) |
| TEMPLATE_PATHS    | Path to custom message templates, default template is `./default.tmpl`, in docker - `/templates/default.tmpl` |

#### Authentication

Additional users may be allowed to command the bot by giving multiple instances
of the `--telegram.admin` command line option or by specifying a
newline-separated list of telegram user IDs in the `TELEGRAM_ADMIN` environment
variable.
```
- TELEGRAM_ADMIN="**********\n************"
--telegram.admin=1 --telegram.admin=2
```
#### Alertmanager Configuration

Now you need to connect the Alertmanager to send alerts to the bot.  
A webhook is used for that, so make sure your `LISTEN_ADDR` is reachable for the Alertmanager.

For example add this to your `alertmanager.yml` configuration:
```yaml
receivers:
- name: 'alertmananger-bot'
  webhook_configs:
  - send_resolved: true
    url: 'http://alertmanager-bot:8080'
```

## Development

Get all dependencies. We use [golang/dep](https://github.com/golang/dep).  
Fetch all dependencies with:

```
dep ensure -v -vendor-only
```

Build the binary using `make`:

```
make install
```

In case you have `$GOPATH/bin` in your `$PATH` you can now simply start the bot by running:

```bash
alertmanager-bot
```

## Missing

##### Commands

* `/silence` - show a specific silence  
* `/silence_del` - delete a silence by command  
* `/silence_add` - add a silence for a alert by command

##### More Messengers

At the moment I only implemented Telegram, because it's so freakin' easy to do.

Messengers considered to add in the future:

* [Slack](https://slack.com/)
* [Mattermost](https://about.mattermost.com/)
* [Matrix](https://matrix.org/)

If one is missing for you just open an issue.
