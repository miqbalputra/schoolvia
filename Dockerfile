# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/schoolvia .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 schoolvia
WORKDIR /srv
COPY --from=build /out/schoolvia /srv/schoolvia
COPY frontend /srv/frontend
USER schoolvia
ENV PORT=8080 \
    FRONTEND_DIR=/srv/frontend
EXPOSE 8080
ENTRYPOINT ["/srv/schoolvia"]
