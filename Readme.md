# Project Name

A Reverse Proxy with Automatic Docker Container Creation for CTF Challenges

## Description

This project is a reverse proxy written in Golang that automatically creates a Docker container for each session. It is designed to support CTF (Capture The Flag) challenges where containers may be broken or compromised, requiring the creation of new containers for each user.

## Features

- Reverse proxy functionality
- Automatic Docker container creation for each session
- Session identifier based on a specific header
- Supports the Docker Compose file specification to create containers for each session

## Installation

1. Clone the repository:

    ```shell
    git clone https://github.com/mart123p/ctf-reverseproxy.git
    ```

2. Build the project:

    ```shell
    go build
    ```

3. Run the executable:

    ```shell
    ./reverse-proxy
    ```

## Usage

1. Set the required environment variables:

    ```shell
    export SESSION_HEADER=<header-name>
    export DOCKER_IMAGE=<docker-image-name>
    ```

2. Start the reverse proxy:

    ```shell
    ./reverse-proxy
    ```

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request.

## License

This project is licensed under the [MIT License](LICENSE).
