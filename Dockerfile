ARG GO_VERSION=1.16
ARG USER=westack

FROM golang:${GO_VERSION}-alpine AS builder

ARG USER

# Install ssl certs
RUN apk add --no-cache --update \
    ca-certificates \
    tzdata \
    && update-ca-certificates

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cli ./cli
COPY cli-utils ./cli-utils
COPY westack ./westack
COPY *.go ./

RUN CGO_ENABLED=0 go build -o /go/bin/westack

# Create a non-root user
RUN adduser -D ${USER}

FROM alpine:3.17

COPY --from=builder /go/bin/westack /usr/local/bin/westack

# Install ssl certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create a non-root user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

USER ${USER}

HEALTHCHECK --interval=5s --timeout=3s \
  CMD wget -q http://localhost:8023/health || exit 1

CMD ["westack server start"]
