# Start from the latest golang base image
FROM golang:latest as builder

# Add Maintainer Info
LABEL maintainer="Honza Pokorny <honza@pokorny.ca>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN make

######## Start a new stage from scratch #######
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/include include
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/smithy .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./smithy"]
