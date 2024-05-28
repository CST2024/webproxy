FROM golang:alpine

WORKDIR /app
COPY . /app

RUN go build -o /app/main

EXPOSE 8000

CMD ["/app/main"]
