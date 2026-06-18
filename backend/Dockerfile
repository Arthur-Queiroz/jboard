# syntax=docker/dockerfile:1
# check=error=true

# Build stage — pgx é pure-Go, então CGO desligado gera binário estático.
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd/server

# Runtime — distroless nonroot, só o binário.
FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
