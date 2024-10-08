FROM golang:1.23-alpine

RUN mkdir /app

WORKDIR /app
COPY ./app /app

RUN go install github.com/air-verse/air@latest

RUN go mod download
RUN go mod tidy

CMD ["air", "-c", ".air.toml"]