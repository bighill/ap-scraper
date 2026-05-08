# build

FROM golang:1.25-alpine AS build

WORKDIR /src/server

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/. .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/apnews-server .

# run

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app/server

COPY --from=build /out/apnews-server /app/server/apnews-server
COPY web /app/web

EXPOSE 9191

ENTRYPOINT ["/app/server/apnews-server"]
