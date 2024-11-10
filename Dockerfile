# Use an official C++ image as a base image
FROM ubuntu:20.04

# Install build dependencies
RUN apt-get update && \
    apt-get install -y \
    cmake \
    g++ \
    git \
    build-essential

# Set the working directory in the container to /app
WORKDIR /app

# Copy the local project files into the /app directory in the container
COPY . /app

# Verify that CMakeLists.txt exists in the container
RUN ls -l /app/CMakeLists.txt

# Run cmake and make to build the project
RUN cmake . && make

# Command to run your application or tests (this can be adjusted depending on your project)
CMD ["./your_executable"]
