FROM golang:1.18-alpine

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN go build -o cli cmd/cli/main.go

CMD [ "./cli" ]