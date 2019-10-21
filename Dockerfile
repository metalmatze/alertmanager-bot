FROM alpine:latest as alpine

# Install the latest SSL certificates
RUN apk add --no-cache ca-certificates

# Create a non privileged user
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

# Start new scratch built for security and minimal footprint
FROM scratch
ENV TEMPLATE_PATHS=/templates/default.tmpl

# Add SSL certificates from the previous alpine stage
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy the unprivileged user
COPY --from=alpine /etc_passwd /etc/passwd

# Copy default template and alertmanager-bot binary
# NOTE: When adding a copy here, don't forget to add it in the .dockerignore
COPY ./default.tmpl ${TEMPLATE_PATHS}
COPY ./alertmanager-bot /usr/bin/alertmanager-bot

EXPOSE 8080
ENTRYPOINT ["/usr/bin/alertmanager-bot"]
USER nobody
