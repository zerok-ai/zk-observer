FROM golang:1.18-alpine
WORKDIR /zk
COPY zk-otlp-receiver .
EXPOSE 2345
CMD ["/zk/zk-otlp-receiver","-c","/zk/config/otlp-config.yaml"]