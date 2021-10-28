FROM golang:1.17

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main ./cmd/main/main.go


RUN chmod +x ./main

EXPOSE 8080

CMD ["./main"]
