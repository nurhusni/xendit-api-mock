FROM golang:1.22-alpine AS build

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o xendit-api-mock ./

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/xendit-api-mock ./xendit-api-mock
EXPOSE 8080
CMD ["/app/xendit-api-mock"]
