FROM golang:alpine AS build
RUN apk --no-cache add gcc g++ make git
WORKDIR /app
COPY . .
RUN go mod init github.com/rustingoff/excel
RUN go mod tidy
RUN GOOS=linux go build -ldflags="-s -w" -o ./bin/web-app ./app/main.go

FROM alpine:3.14
RUN apk --no-cache add ca-certificates
WORKDIR /usr/bin
COPY --from=build /app/bin /bin
EXPOSE 80
ENTRYPOINT /go/bin/web-app --port 80