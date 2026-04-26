#1 этап  
FROM golang:1.26.1 AS builder

WORKDIR /app

COPY . .

RUN go mod tidy && go build -o app ./cmd/main.go

#2 этап финальный
FROM alpine:latest
# не знаю на сколько сильно это потребуется
RUN apk --no-cache add ca-certificates 

WORKDIR /app

COPY --from=builder /app/app .

RUN mkdir -p /app/uploads/

CMD ["./app"]

