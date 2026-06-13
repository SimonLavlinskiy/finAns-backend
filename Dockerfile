FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/finans-api ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/finans-api /finans-api

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/finans-api"]
