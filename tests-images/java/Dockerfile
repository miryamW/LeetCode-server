# Use the official Maven image with OpenJDK 17 (slim version)
FROM maven:3.8.5-openjdk-17-slim

# Set the working directory inside the container to /app
WORKDIR /app

# Copy the pom.xml file into the container's /app directory
COPY pom.xml .

# Run Maven to clean and install the project dependencies
RUN mvn clean install

# Keep the container running by tailing the /dev/null file (useful for debugging or keeping the container alive)
CMD ["tail", "-f", "/dev/null"]
