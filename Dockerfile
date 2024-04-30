FROM golang:1.21.4 as builder

WORKDIR /util

# Dependencies in a layer
COPY go.mod go.sum .
RUN go mod download

COPY . /util
RUN CGO_ENABLED=0 go build -v -o util .

FROM gcr.io/distroless/static
COPY --from=builder /util/util /util
ENTRYPOINT ["/util"]
