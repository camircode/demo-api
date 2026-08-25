# syntax=docker/dockerfile:1

# Multi-stage so the toolchain never reaches the running image. A Go compiler in
# production is 400 MB of attack surface that does nothing.
FROM golang:1.24-alpine AS build

WORKDIR /src

# Dependencies first: this layer is cached unless go.mod or go.sum change, so
# ordinary code changes do not re-download the module graph.
COPY go.mod go.su[m] ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown

# CGO off produces a static binary, which is what lets the final stage be
# distroless rather than a distribution with a libc in it.
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/api ./cmd/api

# `static` carries no shell, no package manager and no libc. Anyone who gets
# code execution in this container arrives somewhere with nothing to use.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/api /api

# Matches runAsUser in the Deployment. Declaring it here as well means the image
# is safe to run without a securityContext, rather than depending on one.
USER 65532:65532

EXPOSE 8080
ENTRYPOINT ["/api"]
