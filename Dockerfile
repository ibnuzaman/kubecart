# syntax=docker/dockerfile:1

# One Dockerfile for all services; pick one with --build-arg SERVICE=<dir in cmd/>.

# Build stage: the full Go toolchain. It runs on the build machine's own
# platform and cross-compiles for the target platform, so there is no slow
# CPU emulation (arm64 Mac -> arm64 image natively, or -> amd64 if asked).
FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS build
ARG TARGETOS TARGETARCH
ARG SERVICE
WORKDIR /src
COPY . .
# Cache mounts keep downloaded modules and compiled packages between builds,
# so a code change doesn't re-download every dependency.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    : "${SERVICE:?build arg SERVICE is required}" && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/$SERVICE

# Final stage: distroless static has no shell or package manager, only CA
# certs, tzdata and a passwd entry for the nonroot user (uid 65532).
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
