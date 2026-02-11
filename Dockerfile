# ---- Build Stage ----
FROM golang:1.24.3-bookworm AS build

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o ts2chapter

# ---- Runtime Stage ----
FROM alpine:3.21

RUN apk add --no-cache ffmpeg

COPY --from=build /build/ts2chapter /usr/local/bin/ts2chapter

ENTRYPOINT ["/usr/local/bin/ts2chapter"]
