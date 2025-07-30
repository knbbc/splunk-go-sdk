FROM golang:1.24.5-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o app .

FROM golang:1.24.5 AS Final

WORKDIR /app
COPY --from=builder /app/app .

# Create a non-root user and switch to it
RUN useradd -m appuser
USER appuser

CMD ["./app"]