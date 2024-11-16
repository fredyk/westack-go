ARG GO_VERSION=1.16
ARG USER

FROM golang:1.22-alpine AS builder

ARG USER=westack

# Install ssl certs
RUN apk add --no-cache --update \
    ca-certificates \
    tzdata \
    && update-ca-certificates

WORKDIR /app

COPY v2/go.mod ./
COPY v2/go.sum ./
RUN go mod download

COPY v2/cli ./cli
COPY v2/cli-utils ./cli-utils
COPY westack ./westack
COPY *.go ./

RUN CGO_ENABLED=0 go build -o /go/bin/westack

# Create a non-root user
RUN adduser -D ${USER}
RUN chmod +x /go/bin/westack

FROM alpine:3.17

ARG USER=westack

COPY --from=builder /go/bin/westack /usr/local/bin/westack

# Install ssl certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create a non-root user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

USER ${USER}

WORKDIR /home/${USER}

# ENV PATH "/usr/local/bin:${PATH}"

HEALTHCHECK --interval=5s --timeout=3s \
  CMD wget -q http://localhost:8023/health || exit 1

CMD ["/usr/local/bin/westack server start"]
