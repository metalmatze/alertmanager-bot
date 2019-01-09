FROM golang:latest

COPY ./default.tmpl /templates/default.tmpl
# Set go bin which doesn't appear to be set already.
ENV GOBIN /go/bin

# build directories
RUN mkdir /app
RUN mkdir -p /go/src/github.com/vu-long/alertmanager-bot
ADD . /go/src/github.com/vu-long/alertmanager-bot
WORKDIR /go/src/github.com/vu-long/alertmanager-bot

# Go dep!
RUN go get -u github.com/golang/dep/...
RUN dep ensure -v -vendor-only

RUN make build

EXPOSE 8080:8080

ENTRYPOINT ["/go/src/github.com/vu-long/alertmanager-bot/alertmanager-bot"]
