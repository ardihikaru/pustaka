# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY docs ./docs
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pustaka .

# Alpine keeps BusyBox wget available for the Compose health check while still
# running the application as an unprivileged user.
FROM alpine:3.21
RUN adduser -D -H -u 10001 pustaka
COPY --from=build /out/pustaka /usr/local/bin/pustaka
COPY --from=build /src/docs /docs
USER pustaka
EXPOSE 8080
ENTRYPOINT ["pustaka", "serve", "/docs", "--addr", ":8080", "--prod"]
