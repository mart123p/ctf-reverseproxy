# Project Name

A Reverse Proxy with Automatic Docker Container Creation for CTF Challenges

## Description

This project is a reverse proxy written in Golang that automatically creates a Docker container for each session. It is designed to support CTF (Capture The Flag) challenges where containers may be broken or compromised, requiring the creation of new containers for each user.

## Features

- Reverse proxy functionality
- Automatic Docker container creation for each session
- Session identifier based on a specific header
- Supports the Docker Compose file specification to create containers for each session

## Usage

Build the container with the help of the docker-compose file:

```bash
docker-compose up --build
```

Create a docker-compose file with the following content:

```yaml
version: '3'
services:
  web:
    annotations:
      ctf-reverseproxy: true
    image: nginx:alpine
    ports:
      - "8080:80"
  web2:
    image: nginx:alpine
    ports:
      - "8081:80"
```

Notice the ctf-reverseproxy annotation. This is used to identify which service should be proxied by the reverse proxy. It can only be set on one service. Currently the reverse proxy does not support volumes. It is recommended to harden the services in docker compose so attendees don't exhaust the resources of the host.

You can use the config file config-example.yaml as a template to create your own config file. The config file should be placed in the same directory as the docker-compose file.

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request.

## License

This project is licensed under the [MIT License](LICENSE).
