# Reading DHT22 Sensor with Raspberry Pi on BalenaOS

This repository contains a Go application for reading DHT22 sensor data using a Raspberry Pi Zero running BalenaOS. The application reads temperature, humidity, and pressure data, and publishes it via MQTT.

## Table of Contents

- [Reading DHT22 Sensor with Raspberry Pi on BalenaOS](#reading-dht22-sensor-with-raspberry-pi-on-balenaos)
  - [Table of Contents](#table-of-contents)
  - [Requirements](#requirements)
  - [Installation](#installation)
  - [Configuration](#configuration)
  - [Running the Application](#running-the-application)
    - [Using Balena to Run on Raspberry Pi](#using-balena-to-run-on-raspberry-pi)
    - [Running Locally](#running-locally)
  - [Shutdown](#shutdown)
  - [License](#license)

## Requirements

- Raspberry Pi with BalenaOS
- DHT22 Module
- Go 1.18+
- Configured AWS IoT Core broker
- `.env` file with necessary configurations
- Docker

## Installation

1. **Clone the repository:**

    ```sh
    git clone https://github.com/lilanga/aws-iot-dht22-sensor.git
    cd aws-iot-dht22-sensor
    ```

2. **Ensure you have Go installed:**

    Follow the instructions on the [official Go website](https://golang.org/doc/install) to install Go.

3. **Install dependencies:**

    This project uses a `go.mod` file to manage dependencies. Ensure you are in the project directory and run:

    ```sh
    go mod tidy
    ```

4. **Create a `.env` file:**

    ```sh
    touch .env
    ```

    Populate the `.env` file with the following variables:

    ```env
    AWS_BROKER=tcps://<YOUR-IOT-CORE-BROKER-URL>:8883/mqtt
    AWS_TOPIC=<YOUR-IOT-TOPIC-ADDED-IN-PERMISSIONS>
    AWS_CLIENT_ID=<YOUR-CLIENT-ADDED-IN-PERMISSIONS>
    REFRESH_INTERVAL=60
    ID=DHT-22-01
    ```

## Configuration

- `ID`: A unique ID for your sensor, Can be anything. I am using DHT-22-01 ID to fetch data from lambda.
- `AWS_BROKER`: Configured AWS IoT Core thing MQTT broker.
- `AWS_TOPIC`: The MQTT topic to publish sensor data to. This need to be mapped in your IoT Core permission profile.
- `AWS_CLIENT_ID`: The Client ID of the MQTT client. This also needs to be mentioned in your IoT Core permission profile.
- `REFRESH_INTERVAL`: The interval (in seconds) at which sensor data is read and published.

## Running the Application

### Using Balena to Run on Raspberry Pi

1. **Install Balena CLI:**

    Follow the instructions on the [official Balena CLI documentation](https://www.balena.io/docs/reference/cli/#installation) to install the Balena CLI.

2. **Log in to Balena:**

    ```sh
    balena login
    ```

3. **Initialize the project:**

    ```sh
    balena push <your-app-name>
    ```

    Replace `<your-app-name>` with the name of your Balena application. This command will build and deploy the application to your Raspberry Pi.
    You need to have a Balena application set up with a Raspberry Pi device before running this command to deploy the application.

### Running Locally

1. **Build and run the application using Docker:**

    Make sure Docker is installed and running on your machine. Then, from the project directory, run:

    ```sh
    docker build -t weather-service .
    docker run --env-file .env -p 8080:8080 weather-service
    ```

2. **Configure Environment Variables:**

    Set the `AWS_TOPIC` and `AWS_CLIENT_ID` to correct values from your AWS IoT Core configurations. Otherwise you will not be able to establish connection to the broker.

## Shutdown

The application is designed to handle graceful shutdown upon receiving an interrupt signal (e.g., `Ctrl+C` or a termination signal).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
