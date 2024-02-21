FROM golang:1.22.0-bookworm

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o ./build/arseeding ./cmd

EXPOSE 8080

CMD [ "./build/arseeding" ]

