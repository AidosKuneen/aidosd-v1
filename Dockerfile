FROM golang:1.15-alpine AS builder
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

# Build the application. Read more about -ldflags here: https://stackoverflow.com/questions/22267189/what-does-the-w-flag-mean-when-passed-in-via-the-ldflags-option-to-the-go-comman
RUN go build -ldflags="-w -s" -o adkd .

WORKDIR /dist
RUN cp /build/adkd .



FROM scratch
# For aidosd.conf, aidosd.db and aidosd.log...
VOLUME /app
COPY --from=builder /dist/adkd /
WORKDIR /app
EXPOSE 8332
ENTRYPOINT ["/adkd"]
