FROM alpine:latest
RUN apk add --update ca-certificates

ADD ./alertmanager-telegram /usr/bin/alertmanager-telegram

EXPOSE 8080

ENTRYPOINT ["/usr/bin/alertmanager-telegram"]
