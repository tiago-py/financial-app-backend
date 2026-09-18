FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /financial-api ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /financial-api /usr/local/bin/financial-api
USER app
EXPOSE 8080
ENTRYPOINT ["financial-api"]
