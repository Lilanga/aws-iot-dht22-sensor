package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/MichaelS11/go-dht"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/joho/godotenv"

	_ "embed"
)

type SensorData struct {
	Humidity    string `json:"humidity"`
	Temperature string `json:"temperature"`
	Pressure    string `json:"pressure"`
	SensorID    string `json:"sensor_id"`
	Timestamp   string `json:"timestamp"`
}

type App struct {
	sensor       *dht.DHT
	sensorID     string
	currentData  SensorData
	interval     time.Duration
	awsIoTClient mqtt.Client
}

var awsBrokerURL = os.Getenv("AWS_BROKER")
var topic = os.Getenv("AWS_TOPIC")
var awsClientID = os.Getenv("AWS_CLIENT_ID")
var sensorID = os.Getenv("ID")

const (
	qos             = 0
	rootCAPath      = "/app/cert/root-CA.crt"
	certificatePath = "/app/cert/cert.pem"
	privateKeyPath  = "/app/cert/private.key"
)

func NewApp() (*App, error) {
	loadEnvVariables()

	if err := initializeHardware(); err != nil {
		return nil, fmt.Errorf("failed to initialize hardware: %w", err)
	}

	sensor, err := dht.NewDHT("GPIO2", dht.Celsius, "DHT22")
	if err != nil {
		return nil, fmt.Errorf("error creating DHT sensor: %w", err)
	}

	interval := getRefreshInterval()

	fmt.Println("Setting up MQTT client...")

	fmt.Println("Setting up AWS IoT client...")
	awsIoTClient, err := setupAWSIoT()
	if err != nil {
		return nil, fmt.Errorf("failed to setup AWS IoT: %w", err)
	}

	return &App{
		sensor:       sensor,
		sensorID:     sensorID,
		interval:     time.Duration(interval) * time.Second,
		awsIoTClient: awsIoTClient,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	fmt.Println("Starting sensor data collection...")
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	fmt.Println("Starting sensor data publishing...")
	go a.publishSensorData(ctx, ticker)

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	// Disconnect from AWS IoT Core
	a.awsIoTClient.Disconnect(250)

	return nil
}

func loadEnvVariables() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("No .env file found: %v", err)
	}
}

func getRefreshInterval() int {
	interval, err := strconv.Atoi(os.Getenv("REFRESH_INTERVAL"))
	if err != nil {
		return 30 // Default to 30 seconds if not set or invalid
	}
	return interval
}

func initializeHardware() error {
	if err := dht.HostInit(); err != nil {
		return fmt.Errorf("host initialization failed: %w", err)
	}
	return nil
}

func loadTLSConfig() (*tls.Config, error) {
	// Load root CA certificate
	rootCA, err := os.ReadFile(rootCAPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load root CA certificate: %w", err)
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair(certificatePath, privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate and key: %w", err)
	}

	// Create a certificate pool for the root CA
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(rootCA); !ok {
		return nil, fmt.Errorf("failed to append root CA certificate")
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
		ClientCAs:    nil,
		RootCAs:      rootCAs,
	}

	return tlsConfig, nil
}

func setupAWSIoT() (mqtt.Client, error) {
	// Load TLS configuration
	tlsConfig, err := loadTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	// Create AWS MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(awsBrokerURL).
		SetClientID(awsClientID).
		SetTLSConfig(tlsConfig).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetKeepAlive(30 * time.Second).
		SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
			fmt.Printf("TOPIC: %s\n", msg.Topic())
			fmt.Printf("MSG: %s\n", msg.Payload())
		}).
		SetOnConnectHandler(func(client mqtt.Client) {
			fmt.Println("Connected to AWS IoT Core")
		}).
		SetConnectionLostHandler(func(client mqtt.Client, err error) {
			fmt.Printf("Connection lost: %v\n", err)
		})

	// Connect to the broker
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	return client, nil
}

func (a *App) publishSensorData(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			humidity, temperature, err := a.sensor.ReadRetry(11)
			if err != nil {
				log.Printf("Read error: %v", err)
				continue
			}

			a.currentData = SensorData{
				Humidity:    fmt.Sprintf("%v", humidity),
				Temperature: fmt.Sprintf("%v", temperature),
				Pressure:    "0",
				SensorID:    a.sensorID,
				Timestamp:   time.Now().Format(time.RFC3339),
			}

			jsonData, err := json.Marshal(a.currentData)
			if err != nil {
				log.Printf("Error marshalling JSON: %v", err)
				continue
			}

			// Publish to AWS IoT Core
			token := a.awsIoTClient.Publish(topic, qos, false, jsonData)
			token.Wait()
			if token.Error() != nil {
				log.Fatalf("failed to publish message: %v", token.Error())
			} else {
				log.Printf("Successfully published message to topic: %s", topic)
			}

		case <-ctx.Done():
			return
		}
	}
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	if err := app.Run(ctx); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
