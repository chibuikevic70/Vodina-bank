FROM golang:1.25-alpine AS build

WORKDIR /app

COPY backend/go.mod ./
RUN go mod download

COPY backend/ ./backend/

WORKDIR /app/backend

RUN go build -o /vodina ./cmd/api/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=build /vodina /vodina
COPY backend/migrations /migrations

EXPOSE 8080

CMD ["/vodina"]
