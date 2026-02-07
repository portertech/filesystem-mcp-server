# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o filesystem ./cmd/filesystem

# Final stage
FROM scratch

COPY --from=builder /build/filesystem /usr/local/bin/filesystem

ENTRYPOINT ["/usr/local/bin/filesystem"]
