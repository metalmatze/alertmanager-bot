FROM alpine:latest
RUN apk add --update ca-certificates

ADD $GOPATH/bin/alertmanager-telegram /usr/bin/

EXPOSE 8080

ENTRYPOINT ["/usr/bin/alertmanager-telegram"]
