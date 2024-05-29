# build stage
FROM golang:1.20.10-alpine3.18 AS builder

LABEL stage=dcron-intermediate

ENV GO111MODULE=on

WORKDIR /go/src/dcron
ADD . .

RUN go build -mod vendor -o /dcron

# final stage
FROM alpine:3.18.4

RUN apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/Asia/Taipei /etc/localtime \
    && echo "Asia/Taipei" > /etc/timezone

WORKDIR /app
COPY --from=builder /dcron /app/dcron
COPY --from=builder /go/src/dcron/config.yaml /app/config.yaml

CMD ["./dcron"]
