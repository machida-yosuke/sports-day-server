FROM golang:1.22-alpine

RUN mkdir /app

WORKDIR /app
COPY . /app

RUN go mod tidy

CMD ["go", "run", "main.go"]