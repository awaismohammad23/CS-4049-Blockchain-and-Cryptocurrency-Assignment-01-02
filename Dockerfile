# Use the official Go image as the base image
FROM golang:1.17

# Set the working directory inside the container
WORKDIR /app

# Copy the Go program source code into the container
COPY . .

# Build the Go program
RUN go build -o myapp

# Expose the port on which your Go program listens
EXPOSE 3000

# Command to run the Go program
CMD ["./myapp"]
