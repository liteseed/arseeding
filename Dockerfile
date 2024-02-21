FROM golang:1.22.0-bookworm

WORKDIR /app

COPY go.mod ./

RUN go mod download
RUN go build -o ./build/arseeding ./cmd

EXPOSE 8080

CMD [ "./build/arseeding" ]

