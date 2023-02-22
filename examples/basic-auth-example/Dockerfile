FROM golang:1.19-alpine as builder

# Prepare for app
WORKDIR /usr/src/app

# COPY dependencies definition
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the code
COPY ./server ./server
COPY ./common ./common

# Get the dependencies
RUN go get -d -v ./server

# Build the code
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/westack_basic_auth ./server/

# Make binary executable
RUN chmod +x /go/bin/westack_basic_auth

FROM scratch

# Import from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the code
WORKDIR /usr/src/app
COPY --from=builder /go/bin/westack_basic_auth .
COPY ./common/ ./common/
COPY ./server/datasources*json ./server/
COPY ./server/model-config.json ./server/model-config.json

EXPOSE 8023

ENTRYPOINT [ "./westack_basic_auth" ]
