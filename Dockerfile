FROM oven/bun:1 AS client
WORKDIR /src/app
COPY app/package.json app/bun.lock ./
RUN bun install --frozen-lockfile
COPY app/ .
RUN bun run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=client /src/app/dist/ ./app/dist/
RUN CGO_ENABLED=0 go build -o /herald ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /herald /usr/local/bin/herald
EXPOSE 8080
ENTRYPOINT ["herald"]
