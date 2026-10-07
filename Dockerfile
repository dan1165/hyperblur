# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS TARGETARCH
RUN apk add --no-cache ca-certificates

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /hyperblur ./cmd/hyperblur

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /hyperblur /hyperblur

EXPOSE 8000
USER 65534:65534
ENTRYPOINT [ "/hyperblur" ]
