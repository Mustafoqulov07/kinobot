# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Keshdan foydalanish uchun dependency'larni ko'chirib olamiz
COPY go.mod go.sum ./
RUN go mod download

# Kodlarni ko'chirib, binar faylni quramiz
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o kinobot .

# Final stage (juda kichik va yengil alpine konteyner)
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/kinobot .
COPY --from=builder /app/static ./static

ENV PORT=8000
EXPOSE 8000

CMD ["./kinobot"]
