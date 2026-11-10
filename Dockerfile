FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /gateway ./cmd/gateway

FROM alpine:3.21
RUN addgroup -S gateway && adduser -S -G gateway gateway
COPY --from=build /gateway /usr/local/bin/gateway
USER gateway
EXPOSE 8080
ENV PLOOS_GATEWAY_LISTEN=0.0.0.0:8080
ENTRYPOINT ["/usr/local/bin/gateway"]
