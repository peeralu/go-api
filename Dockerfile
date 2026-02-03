# ---------- build stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# copy go mod first (cache friendly)
COPY go.mod go.sum ./
RUN go mod download

# copy source
COPY . .

# build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o app

# ---------- runtime stage ----------
FROM alpine:3.20

WORKDIR /app

# ca-cert for https
RUN apk add --no-cache ca-certificates

# copy binary only
COPY --from=builder /app/app .

# expose port (ปรับตาม app)
EXPOSE 8080

# run
CMD ["./app"]
