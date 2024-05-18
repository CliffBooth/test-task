FROM golang:1.22
WORKDIR /app
COPY . .
RUN go build -o main .
ENTRYPOINT ["/app/main"]