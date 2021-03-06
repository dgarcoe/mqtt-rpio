package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	rpio "github.com/stianeikeland/go-rpio"
)

var (
	mqttBroker = flag.String("mqttBroker", "", "MQTT broker URI (mandatory). E.g.:192.168.1.1:1883")
	topic      = flag.String("topic", "", "Topic where hub-ctrl messages will be received (mandatory)")
	user       = flag.String("user", "", "MQTT username")
	pwd        = flag.String("password", "", "MQTT password")
)

//Message Used to hold MQTT JSON messages
type Message struct {
	Type  string
	GPIO  int
	Mode  string
	Level string
}

type GPIO struct {
	Pin   rpio.Pin
	Mode  string
	Level string
}

var gpioList map[int]GPIO

//Connect to the MQTT broker
func connectMQTT() (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker("tcp://" + *mqttBroker)

	if *user != "" && *pwd != "" {
		opts.SetUsername(*user).SetPassword(*pwd)
	}

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("%s", token.Error())
	}

	return client, nil
}

//Callback for MQTT messages received through the subscribed topic
func mqttCallback(client mqtt.Client, msg mqtt.Message) {

	var jsonMessage Message
	var gpioData GPIO

	log.Printf("Message received: %s", msg.Payload())

	err := json.Unmarshal(msg.Payload(), &jsonMessage)
	if err != nil {
		log.Printf("Error parsing JSON: %s", err)
	}

	typeMsg := jsonMessage.Type

	switch typeMsg {
	case "GPIOSetMode":
		log.Printf("GPIO %d setting mode to %s ", jsonMessage.GPIO, jsonMessage.Mode)
		gpio := jsonMessage.GPIO
		gpioData.Mode = jsonMessage.Mode
		gpioData.Pin = rpio.Pin(gpio)
		if gpioData.Mode == "Output" {
			gpioData.Pin.Output()
		} else if gpioData.Mode == "Input" {
			gpioData.Pin.Input()
		}
		gpioList[gpio] = gpioData
	case "GPIOLevel":
		log.Printf("GPIO %d setting level to %s", jsonMessage.GPIO, jsonMessage.Level)
		gpio := jsonMessage.GPIO
		gpioData = gpioList[gpio]
		gpioData.Level = jsonMessage.Level
		if gpioData.Level == "High" {
			gpioData.Pin.High()
		} else if gpioData.Level == "Low" {
			gpioData.Pin.Low()
		}
	}

}

func init() {
	flag.Parse()
	gpioList = make(map[int]GPIO)
}

func main() {

	//Check command line parameters
	if *mqttBroker == "" || *topic == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	//Channel used to block while receiving messages
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	clientMQTT, err := connectMQTT()
	if err != nil {
		log.Fatalf("Error connecting to MQTT broker: %s", err)
	}

	log.Printf("Connected to MQTT broker at %s", *mqttBroker)

	if token := clientMQTT.Subscribe(*topic, 0, mqttCallback); token.Wait() && token.Error() != nil {
		log.Fatalf("Error subscribing to topic %s : %s", *topic, err)
	}

	log.Printf("Subscribed to topic %s", *topic)

	err = rpio.Open()
	if err != nil {
		log.Fatalf("Couldn't open GPIO: %s", err)
	}

	<-c

	rpio.Close()

}
