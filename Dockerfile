FROM golang:1.18-alpine
WORKDIR /zk
COPY zk-otlp-receiver .
CMD ["/zk/zk-otlp-receiver","-c","/zk/config/otlp-config.yaml"]