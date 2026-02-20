FROM golang:1.24.0 AS builder

WORKDIR /util

# Dependencies in a layer
COPY go.mod go.sum .
RUN go mod download

COPY . /util
RUN CGO_ENABLED=1 go build -v -o util .

FROM golang:1.21.4
COPY --from=builder /util/util /util
ENTRYPOINT ["/util"]
