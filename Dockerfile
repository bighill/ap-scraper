# build

FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/apnews-server ./cmd/server

# run

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /out/apnews-server /app/apnews-server
COPY web /app/web

EXPOSE 9191

ENTRYPOINT ["/app/apnews-server"]
