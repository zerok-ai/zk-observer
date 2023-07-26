# Dockerfile

# Use Alpine as the base image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

COPY zk-otlp-receiver /app/myapp

# Set executable permissions for the Go executable
RUN chmod +x /app/myapp

# Run the Go executable
CMD ["./myapp","-c","/opt/otlp-config.yaml"]