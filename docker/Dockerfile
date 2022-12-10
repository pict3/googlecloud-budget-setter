FROM golang:1.16.5

WORKDIR /app

RUN go get github.com/sirupsen/logrus

COPY ./src /app

CMD ["go", "run", "main.go"]
