FROM oven/bun:1 AS web
WORKDIR /src/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ .
RUN bun run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/scalar/scalar.js /src/web/scalar/scalar.js
COPY --from=web /src/web/scalar/scalar.css /src/web/scalar/scalar.css
RUN CGO_ENABLED=0 go build -o /herald ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /herald /usr/local/bin/herald
EXPOSE 8080
ENTRYPOINT ["herald"]
