# trunk-ignore-all(trivy/DS002)
ARG GO_VERSION=1.16

FROM golang:${GO_VERSION}-alpine AS builder

# Install ssl certs
RUN apk add --no-cache --update ca-certificates tzdata && update-ca-certificates

WORKDIR /app

COPY go.mod go.sum./
RUN go mod download

COPY cli ./cli
COPY cli-utils ./cli-utils
COPY westack ./westack
COPY *.go ./

RUN CGO_ENABLED=0 go build -o /go/bin/westack

FROM alpine:latest

COPY --from=builder /go/bin/westack /usr/local/bin/westack

# Install ssl certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

CMD ["westack server start"]

HEALTHCHECK --interval=5s --timeout=3s \
  CMD wget -q http://localhost:8023/health || exit 1
