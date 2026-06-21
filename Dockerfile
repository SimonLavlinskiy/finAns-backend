FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/finans-api ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/finans-migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/finans-api /finans-api
COPY --from=builder /out/finans-migrate /finans-migrate
COPY --from=builder /src/db/migrations /db/migrations

EXPOSE 8082
USER nonroot:nonroot
ENTRYPOINT ["/finans-api"]
