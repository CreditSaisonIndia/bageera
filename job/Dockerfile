# Multi-stage build
FROM golang:1.20-alpine as builder

WORKDIR /app/bageera

# COPY go.mod, go.sum and download the dependencies
COPY go.* /app/bageera/
RUN go mod download

# COPY all files inside the project
COPY . .


# Copy the props directory
COPY ./internal/serviceConfig/props /app/bageera/internal/serviceConfig/props


RUN ls -l /app/bageera/internal/serviceConfig/props
# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bageera/main ./cmd

# Final image
FROM alpine:latest

# Copy the binary from the builder stage
COPY --from=builder /app/bageera/main /app/bageera/main
COPY --from=builder /app/bageera/internal/serviceConfig/props /app/bageera/internal/serviceConfig/props


ENTRYPOINT [ "/app/bageera/main" ]

#docker push public.ecr.aws/y5l1x4y8/test-pq-repo:latest
#docker build  -t test-pq-repo:latest .
#docker tag test-pq-repo:latest public.ecr.aws/y5l1x4y8/test-pq-repo:latest
# aws ecr-public get-login-password --region us-east-1 --profile Development-Technology-Developer-971709774307 | docker login --username AWS --password-stdin public.ecr.aws/y5l1x4y8
