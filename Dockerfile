# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /openblur ./cmd/openblur

FROM alpine:3.21
WORKDIR /openblur

COPY --from=build /openblur /openblur/openblur
COPY assets ./assets

RUN addgroup -g 1000 -S openblur && \
    adduser -u 1000 -S openblur -G openblur && \
    chown -R openblur:openblur /openblur

EXPOSE 8000
USER openblur
ENTRYPOINT [ "/openblur/openblur" ]
