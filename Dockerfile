FROM golang:1.24.11-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/slsbench ./cmd/slsbench

FROM python:3.12-slim-bookworm

WORKDIR /opt/slsbench

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && pip install --no-cache-dir schemathesis \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/slsbench /usr/local/bin/slsbench
COPY scripts ./scripts

ENV SLSBENCH_SCRIPT_DIR=/opt/slsbench/scripts

ENTRYPOINT ["slsbench"]
