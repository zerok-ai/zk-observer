FROM golang:1.18-alpine
WORKDIR /zk
COPY zk-otlp-receiver .
RUN ["go", "install", "github.com/go-delve/delve/cmd/dlv@master"]
CMD ["/zk/zk-otlp-receiver","-c","/zk/config/config.yaml"]