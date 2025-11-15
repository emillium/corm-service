# 1. DECLARE ARG BEFORE FROM: This tells Docker to expect the GO_VERSION argument.
#    We set a default (e.g., 1.22) in case the CI doesn't pass it.
ARG GO_VERSION=1.23

# Stage 1: Build the Go binary
FROM golang:${GO_VERSION}-alpine AS builder

# Set build arguments passed from the CI/CD pipeline
ARG VERSION
ARG TARGETOS
ARG TARGETARCH
ARG SERVICE_NAME

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum to cache dependencies
COPY go.mod go.sum ./
RUN go mod download
RUN go mod vendor

# Copy the rest of the source code
COPY . .

# Build the Go application
# -ldflags="-w -s" removes debug info for a smaller binary.
# -ldflags="-X main.Version=${VERSION}" embeds the Git SHA into the binary.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-w -s -X main.Version=${VERSION}" \
    -o /app/${SERVICE_NAME} \
    ./cmd/${SERVICE_NAME} 

# Stage 2: Create a minimal production image
# Using 'scratch' is the smallest possible base image, containing only the binary.
FROM scratch AS final

ARG SERVICE_NAME 
ENV SERVICE_NAME=${SERVICE_NAME}

# Optional: Use alpine for a slightly larger, but still tiny, base if you need tools like 'ca-certificates'
# FROM alpine:latest AS final
# RUN apk --no-cache add ca-certificates

# Copy the static binary from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/${SERVICE_NAME} /${SERVICE_NAME}

# Expose the port your service listens on (e.g., 8080)
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/${SERVICE_NAME}"]
