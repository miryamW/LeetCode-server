# Leetcode-like System

## Overview

This project implements a system similar to Leetcode.com, allowing users to create, edit, delete, read, and test coding questions - supporting coding in the languages python & java. The system is built with a backend API server written in Go using the Gin framework and a frontend built with Nuxt.js.

## Features

- **Question Management**: Create, retrieve, update, and delete coding questions.
- **Test Solutions**: Submit solutions and run predefined tests to check their correctness.

## Architecture

The system consists of two main components:
1. **Backend**: A RESTful API built with Go and Gin.
2. **Frontend**: A user interface built with Nuxt.js.

### Backend

The backend API provides endpoints for managing coding questions and running tests. It uses Gin for routing and handling HTTP requests.

### Frontend

The frontend is a web application built with Nuxt.js, providing an intuitive interface for interacting with the system.

## Setup

### Prerequisites

- Docker
- Docker Compose

### Installation

1. **Clone the repository**:
   ```sh
   git clone https://github.com/miryamW/LeetCode-server.git
   cd LeetCode-server
   ### Set up environment variables
   
Make sure Kubernetes is enabled in Docker Desktop on your computer.

### Set up environment variables

Create a `.env` file in the root directory and add the necessary environment variables. For example:

```env
DATABASE_URL=Your Database URL in the Mongo container.
DATABASE_NAME=Your database name
COLLECTION_NAME=Your collection name in the DB
KUBE_PATH=Your .kube directory location
```
Make sure to replace the placeholders with your actual values
### Running the application
To start the application locally using Docker Compose, follow these steps:

1.Build and run the application:
 ```sh
docker-compose up --build
```
This command will build the Docker containers (if not already built) and start the application. The backend
system should be accessible at http://localhost:8000 and the frontend system should be accessible at http://localhost:3000.

