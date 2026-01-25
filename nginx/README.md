# Nginx Configuration

This directory contains the nginx configuration files for the GeekBudget application.

## Files

- `nginx.conf` - Main nginx configuration file
- `conf.d/default.conf` - Default server configuration with reverse proxy setup

## Configuration

The nginx server acts as a reverse proxy that:

1. **API Routes** (`/v1/`) - Proxies to the backend Go service on port 8080
2. **Web Routes** (`/web/`) - Proxies to the backend for any web interface routes
3. **Frontend Routes** (`/`) - Proxies to the Angular frontend service on port 4200
4. **Health Check** (`/health`) - Returns a simple health status

## Features

- **SPA Support**: Proper routing for Single Page Applications
- **Security Headers**: X-Frame-Options, X-Content-Type-Options, etc.
- **Gzip Compression**: Enabled for better performance
- **Large File Uploads**: Supports up to 1GB file uploads
- **Dynamic DNS Resolution**: Uses Docker's embedded DNS for service discovery

## SSL/HTTPS

To enable HTTPS, uncomment and configure the SSL server block in `conf.d/default.conf` and provide SSL certificates in the `ssl/` directory.

## Usage

This configuration is automatically used when running the application with Docker Compose:

```bash
make docker-up
```

The application will be available at:
- HTTP: http://localhost
- HTTPS: https://localhost (if SSL is configured)
