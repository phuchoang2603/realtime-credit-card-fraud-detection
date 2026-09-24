# Build one module independently. The build context is the repository root.
# docker build -f infra/docker/go-service.Dockerfile \
#   --build-arg SERVICE_PATH=src/edge --build-arg BUILD_PKG=./cmd/edge \
#   --build-arg BUILD_BIN=edge --build-arg EXPOSE_PORT=8080 -t fraud-edge .
FROM golang:1.26-bookworm AS builder

ARG SERVICE_PATH=src/edge
ARG BUILD_PKG=./cmd/edge
ARG BUILD_BIN=edge
ENV GOWORK=off
WORKDIR /src
COPY ${SERVICE_PATH} ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /out/${BUILD_BIN} ${BUILD_PKG}

FROM gcr.io/distroless/static-debian12:nonroot
ARG BUILD_BIN=edge
ARG EXPOSE_PORT=8080
COPY --from=builder /out/${BUILD_BIN} /app/service
EXPOSE ${EXPOSE_PORT}
USER nonroot:nonroot
ENTRYPOINT ["/app/service"]
