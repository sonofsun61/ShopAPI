FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/shopapi ./cmd/ShopAPI

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

COPY --from=build /out/shopapi /usr/local/bin/shopapi

USER app

EXPOSE 8080

ENTRYPOINT ["shopapi"]
