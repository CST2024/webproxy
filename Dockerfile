FROM golang:alpine

COPY . /app
WORKDIR /app

RUN go build -o main

EXPOSE 8000

CMD ["./main"]
