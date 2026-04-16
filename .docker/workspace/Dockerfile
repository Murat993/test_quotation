FROM golang:1.26.2-bookworm

WORKDIR /workspace

RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.0

COPY go.mod go.sum* ./
RUN go mod download

CMD ["tail", "-f", "/dev/null"]
