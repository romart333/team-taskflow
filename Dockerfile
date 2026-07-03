FROM golang:1.26-alpine AS build

WORKDIR /src
COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ ./
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.21

WORKDIR /srv
COPY --from=build /out/server ./server
COPY app/configs/config.yaml ./configs/config.yaml

EXPOSE 8080
ENTRYPOINT ["/srv/server"]
