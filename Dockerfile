# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /openblur ./cmd/openblur

FROM alpine:3.21
WORKDIR /openblur

COPY --from=build /openblur /openblur/openblur

EXPOSE 8000
USER 65534:65534
ENTRYPOINT [ "/openblur/openblur" ]
