
# Rate Limiter in Go

## Introduction

This project is an implementation of a rate limiter in Go that can be configured to limit the maximum number of requests per second based on a specific IP address or an access token. The rate limiter is designed to be used as middleware in a web server, controlling the traffic and ensuring that clients adhere to specified rate limits.

## Features

- **Middleware Integration**: Easily integrates as middleware in your Go gin http server.
- **Configurable Limits**: Set maximum requests per second via environment variables or a `.env` file.
- **IP and Token-based Limiting**: Limits requests based on IP addresses or access tokens.
- **Custom Block Duration**: Configure how long an IP or token is blocked after exceeding the limit.
- **Redis Backend**: Uses Redis for storing limiter data, ensuring high performance and scalability.
- **Pluggable cache service Strategy**: The cache service mechanism can be swapped out with a different backend by implementing a simple interface.
- **Separation of Concerns**: The rate limiting logic is separated from the middleware for cleaner code management.
