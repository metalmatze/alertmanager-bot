FROM alpine:latest
ENV TEMPLATE_PATHS=/templates/default.tmpl
RUN apk add --update ca-certificates

COPY ./default.tmpl /templates/default.tmpl
COPY ./alertmanager-bot /usr/bin/alertmanager-bot

ENTRYPOINT ["/usr/bin/alertmanager-bot"]
