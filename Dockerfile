# syntax=docker/dockerfile:1

FROM node:22-bookworm-slim AS frontend
WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
# Same-origin API when served by the Go binary.
ENV NUXT_PUBLIC_API_BASE=
RUN npm run generate

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/.output/public ./server/web/dist

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
	--mount=type=cache,target=/go/pkg \
	CGO_ENABLED=0 \
	GOOS=$TARGETOS \
	GOARCH=$TARGETARCH \
	go build -o campfire-export github.com/topi314/campfire-export

FROM alpine

COPY --from=build /build/campfire-export /bin/campfire-export

ENTRYPOINT ["/bin/campfire-export"]

CMD ["-config", "/var/lib/config.toml"]
