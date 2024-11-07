FROM golang:1.22.8 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:3.20.3

RUN apk --no-cache add ca-certificates

WORKDIR /

COPY --from=builder /app/main .

CMD ["./main"]
