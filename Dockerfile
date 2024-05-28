FROM golang:alpine

COPY . /app
WORKDIR /app

RUN go build

EXPOSE 8000

CMD ["./webproxy"]
