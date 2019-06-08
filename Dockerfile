FROM golang:latest

LABEL maintainer="Donny Xia <xiadong.main@gmail.com>"

WORKDIR $GOPATH/src/

ENV GO111MODULE=on

COPY ./go.mod ./go.sum main.go ./
RUN go mod download

RUN CGO_ENABLED=0 go build \
    -o /rss-wechat .

EXPOSE 8081

CMD ["/rss-wechat", "8081"]