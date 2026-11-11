FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/break-daemon ./cmd/daemon
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/break-ctl ./cmd/cli

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/bin/break-daemon /usr/local/bin/break-daemon
COPY --from=builder /app/bin/break-ctl /usr/local/bin/break-ctl

ENV WELLBEING_DB_PATH=/data/wellness_local.db
VOLUME /data

CMD ["break-daemon"]
