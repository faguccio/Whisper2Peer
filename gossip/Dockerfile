FROM golang:1.23 as builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN make build

# Start a new stage from scratch
FROM ubuntu:latest

# Set working directory
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/gossip .

# Document the port numbers your container will expose
EXPOSE 6001/TCP
EXPOSE 7001/TCP

# Set the gossip binary as entrypoint.
# => Command line arguments to docker run <image> will be appended after all elements
ENTRYPOINT ["./gossip"]
