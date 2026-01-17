FROM golang:1.25.6-bookworm as builder
COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod tidy
COPY . .
ENV CGO_ENABLED=0
RUN go build -o /app/operator
FROM scratch
COPY --from=builder /app/operator /operator
ENTRYPOINT ["/operator"]