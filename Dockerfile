# syntax=docker/dockerfile:1.6

FROM golang:1.22-alpine AS builder
WORKDIR /app
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/sql2metrics ./cmd/collector

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /bin/sql2metrics /bin/sql2metrics
COPY configs configs
ENTRYPOINT ["/bin/sql2metrics"]
CMD ["-config", "configs/config.yml"]
