# Use an official Ubuntu as a base image
FROM ubuntu:latest

# Update package lists and install Nginx
# Update package lists and install necessary dependencies
RUN apt-get update && apt-get install -y \
    nginx \
    golang \
    python3 \
    python3-pip \
    curl \
    unzip \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install awscli

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]