FROM golang:latest

COPY ./default.tmpl /templates/default.tmpl
# COPY ./AlertingSystem /usr/bin/AlertingSystem
# Set go bin which doesn't appear to be set already.
ENV GOBIN /go/bin

# build directories
RUN mkdir /app
RUN mkdir /go/src/github.com/vu-long/AlertingSystem
ADD . /go/src/github.com/vu-long/AlertingSystem
WORKDIR /go/src/github.com/vu-long/AlertingSystem

# Go dep!
RUN go get -u github.com/golang/dep/...
RUN dep ensure -v -vendor-only

RUN make

EXPOSE 8080:8080

ENTRYPOINT ["/go/src/github.com/vu-long/AlertingSystem/AlertingSystem"]
