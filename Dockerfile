FROM golang:alpine AS build
RUN apk add --no-cache make git

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make

FROM alpine:latest
ENV TEMPLATE_PATHS=/templates/default.tmpl
RUN apk add --no-cache --update ca-certificates tini

COPY ./default.tmpl /templates/default.tmpl
COPY --from=build /usr/src/app/alertmanager-bot /usr/bin/alertmanager-bot

ENTRYPOINT ["/sbin/tini", "--"]

CMD ["/usr/bin/alertmanager-bot"]
