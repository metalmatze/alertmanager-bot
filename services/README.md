## Prometheus Monitoring

This repository contains minimal Prometheus Server, NodeExporter, BlackBoxExporter, AlertManager and Grafana implementation for monitoring various services. You can use this repository to monitor a bare-metal Linux instance or to monitor Apache, NGINX or other HTTP based services using Prometheus.

## Monitoring a Bare-Metal Linux Server

To monitor a stand-alone Linux Server, you have to checkout against the tag v1.0 of the repository. Where all the configurations for monitoring a stand-alone Linux Server are available. Just `docker-compose up -d` and you're good to go. (You have to map alerts manually against tag v1.0)

## Monitoring HTTP-based Web Services

The v1.1 tag of the repository monitors 2 HTTP-based Web Services by default: An Apache httpd server and NGINX server both running in Docker Containers. If either or both of them goes down, an Prometheus will fire alerts in the form emails specified in the `config.yml` file in the AlertManager folder.