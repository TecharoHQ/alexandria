ARG GO_VERSION=1.24
ARG ALPINE_VERSION=3.22
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/root/go \
  go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/root/go \
  GOOS=${TARGETOS} \
  GOARCH=${TARGETARCH} \
  CGO_ENABLED=0 \
  go build \
  -gcflags "all=-N -l" \
  -o /app/bin/alexandria \
  ./cmd/alexandria

FROM alpine:${ALPINE_VERSION} AS run
WORKDIR /app

RUN apk -U add ca-certificates mailcap

COPY --from=build /app/bin/alexandria /app/bin/alexandria

CMD ["/app/bin/alexandria"]

LABEL org.opencontainers.image.source="https://github.com/TecharoHQ/alexandria"
