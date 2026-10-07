# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /hyperblur ./cmd/hyperblur

FROM alpine:3.21
WORKDIR /hyperblur

COPY --from=build /hyperblur /hyperblur/hyperblur

EXPOSE 8000
USER 65534:65534
ENTRYPOINT [ "/hyperblur/hyperblur" ]
