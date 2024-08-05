#Builder image
FROM golang:alpine3.15 as builder
RUN apk add git
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o weather-service .

#Runner image
FROM alpine:3.15
WORKDIR /app
COPY --from=builder /build/weather-service .
COPY ./cert /app/cert

ENTRYPOINT [ "./weather-service" ]