# Multi-stage build for the Go ranking service. Produces a small static image.
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO disabled for a fully static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/ranking ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/ranking /app/ranking

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/ranking"]
