/*
* Copyright 2023-2025 Thorsten A. Knieling
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
 */

package energymonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"github.com/tknie/log"
	"github.com/tknie/services"
	"golang.org/x/text/message"
)

var mqttCounter = uint64(0)
var mqttDone = make(chan bool, 1)

const DefaultLoopSeconds = 120
const DefaultMaxTries = 120

var OutLoopSeconds = DefaultLoopSeconds
var CloseIfStuck = false

var blockRequestTime time.Time = time.Now().Add(time.Duration(10) * time.Second)
var currentRequested float64 = 0
var currentDelivered float64 = 0

type Mapping []struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Type        string `yaml:"type"`
	IfNegative  string `yaml:"ifNegative,omitempty"`
}

type Topic struct {
	Name       string  `yaml:"name"`
	Mapping    Mapping `yaml:"mapping"`
	pahoClient *paho.Client
}

var pahoClient *paho.Client

func LoopCounterAndCancelOutput(msgChan chan *paho.Publish, topicMap map[string]*Topic) {
	lastCounter := uint64(0)
	lastTime := time.Now()
	try := 0
	services.ServerMessage("Start MQTT analyze loop with output every %d seconds", OutLoopSeconds)
	for {
		select {
		case m := <-msgChan:
			mqttCounter++
			log.Log.Debugf("%s: Message: %s", m.Topic, string(m.Payload))
			if topic, ok := topicMap[m.Topic]; ok {
				x := make(map[string]interface{})
				log.Log.Debugf("EVENT....%s", string(m.Payload))
				err := json.Unmarshal(m.Payload, &x)
				if err != nil {
					fmt.Println("JSON unmarshal fails:", err)
					fmt.Println("JSON unmarshal fails for payload:", string(m.Payload))
					continue
				}

				em := topic.ParseMessage(x)
				if em != nil {
					topic.processEvent(em)
					os.Stdout.Sync()
				}
			}
		case <-mqttDone:
			services.ServerMessage("Ecoflow analyze loop is stopped")
			return
		case <-time.After(time.Second * time.Duration(OutLoopSeconds)):
			if mqttCounter == lastCounter && CloseIfStuck {
				if try > 10 {
					services.ServerMessage("Received MQTT msgs error still stuck")
					os.Exit(10)
				}
				try++
			} else {
				try = 0
			}
			lastCounter = mqttCounter
		}
		if lastTime.Add(60 * time.Second).Before(time.Now()) {
			p := message.NewPrinter(message.MatchLanguage("en"))
			services.ServerMessage(p.Sprintf("Received realtime MQTT msgs: %4d", mqttCounter))
			lastTime = time.Now()
		}
	}
}

func tryConnectMQTT(server string, tries int) net.Conn {
	var err error
	var conn net.Conn
	for count := 0; count < tries; count++ {
		conn, err = net.Dial("tcp", server)
		if err == nil {
			return conn
		}
		if count < tries {
			services.ServerMessage("Error connecting MQTT retrying soon ... %v", err)
			time.Sleep(10 * time.Second)
		} else {
			services.ServerMessage("Error connecting MQTT ... %v", err)
		}
	}
	if err != nil {
		log.Log.Fatalf("Failed to dial to %s: %s", server, err)
	}
	return nil
}

func (config *AdapterConfig) ConnectMQTT(f func(chan *paho.Publish, map[string]*Topic)) {
	if config.Mqtt == nil || config.Mqtt.Server == "" {
		services.ServerMessage("No definition or MQTT server set")
		return
	}
	refreshCurrentPowerRequest()
	if config.Mqtt.MaxTries == 0 {
		config.Mqtt.MaxTries = DefaultMaxTries
	}
	logger := &MQTTWrapperLogger{}
	msgChan := make(chan *paho.Publish)

	if config.Mqtt.LoopIntervalSeconds > 0 {
		OutLoopSeconds = config.Mqtt.LoopIntervalSeconds
	}

	services.ServerMessage("Connect TCP/IP to %s", config.Mqtt.Server)
	conn := tryConnectMQTT(config.Mqtt.Server, config.Mqtt.MaxTries)

	pahoClient = paho.NewClient(paho.ClientConfig{PacketTimeout: 2 * time.Minute,
		Router: paho.NewStandardRouterWithDefault(func(m *paho.Publish) {
			msgChan <- m
		}),
		Conn: conn,
		OnServerDisconnect: func(d *paho.Disconnect) {
			services.ServerMessage("MQTT disconnected: %d - %s", d.ReasonCode, d.Properties.ReasonString)

		},
		OnClientError: func(err error) {
			services.ServerMessage("MQTT client error: %v", err)
			debug.PrintStack()
		},
	})
	pahoClient.SetDebugLogger(logger)
	pahoClient.SetErrorLogger(logger)
	services.ServerMessage("Connecting MQTT paho services to %s", config.Mqtt.Server)
	password := os.ExpandEnv(config.Mqtt.Password)

	// connect to MQTT and listen and subscribe
	cp := &paho.Connect{
		KeepAlive:  30,
		ClientID:   config.Mqtt.Clientid,
		CleanStart: true,
		Username:   config.Mqtt.Username,
		Password:   []byte(password),
	}

	if config.Mqtt.Username != "" {
		cp.UsernameFlag = true
	}
	if password != "" {
		cp.PasswordFlag = true
	}

	// connecting to MQTT server
	ca, err := pahoClient.Connect(context.Background(), cp)
	if err != nil {
		services.ServerMessage("Error to connect paho services to %s with %s: %v",
			config.Mqtt.Server, config.Mqtt.Username, err)
		log.Log.Fatalf("Error to connect to MQTT: %v", err)
	}
	if ca.ReasonCode != 0 {
		services.ServerMessage("Failed to connect paho services to %s with %s with reason code %d",
			config.Mqtt.Server, config.Mqtt.Username, ca.ReasonCode)

		log.Log.Fatalf("Failed to connect to %s : %d - %s", config.Mqtt.Server, ca.ReasonCode, ca.Properties.ReasonString)
	}

	services.ServerMessage("Connecting MQTT to %s", config.Mqtt.Server)

	ic := make(chan os.Signal, 1)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ic
		fmt.Println("signal received, exiting")
		if config != nil {
			d := &paho.Disconnect{ReasonCode: 0}
			pahoClient.Disconnect(d)
		}
		os.Exit(0)
	}()

	topicMap := make(map[string]*Topic)
	// subscribe to a subscription MQTT topic
	subscriptions := make([]paho.SubscribeOptions, 0)
	for _, topic := range config.Mqtt.Topics {
		topic.pahoClient = pahoClient
		topicMap[topic.Name] = topic
		subscriptions = append(subscriptions, paho.SubscribeOptions{Topic: topic.Name,
			QoS: byte(config.Mqtt.Qos)})

		services.ServerMessage("Subscribed MQTT to %s", topic.Name)
	}
	sa, err := pahoClient.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: subscriptions,
	})
	if err != nil {
		services.ServerMessage("Error subscribing MQTT ... %v", err)
		log.Log.Fatalf("Error subscribing MQTT: %v", err)
	}
	if sa.Reasons[0] != byte(config.Mqtt.Qos) {
		log.Log.Fatalf("Failed to subscribe to %v : %d", subscriptions, sa.Reasons[0])
	}

	go f(msgChan, topicMap)
}

func (topic *Topic) processEvent(event map[string]interface{}) {
	log.Log.Debugf("Processing event for topic: %s, got event: %v request: %f",
		topic.Name, event, currentRequested)
	newRequested := currentRequested
	out := event["out"].(float64)
	power := event["power"].(float64)
	log.Log.Debugf("Pre-Power: %f, out: %f, new requested: %f, current requested: %f, blockRequestTime: %v",
		power, out, newRequested, currentRequested, time.Until(blockRequestTime))

	if currentRequested == 0 || !adapter.DefaultConfig.RealtimeRequest {
		refreshCurrentPowerRequest()
		return
	}
	log.Log.Infof("Realtime request = %v  or new requested is same as last requested %f, computed value: %f power: %f out: %f",
		adapter.DefaultConfig.RealtimeRequest, currentRequested, newRequested, power, out)

	if power < 0 && out == 0 {
		out = -power
		power = 0
	}
	log.Log.Debugf("After align Power: %f, out: %f, new requested: %f, current requested: %f",
		power, out, newRequested, currentRequested)

	if power > 0 && blockRequestTime.After(time.Now()) {
		log.Log.Debugf("Blocked until time shift or power 0")
		refreshCurrentPowerRequest()
		return
	}

	switch {
	case out > 0:
		newRequested = float64(currentRequested) - out
		log.Log.Infof("OUT found new requested: %f, current requested: %f",
			newRequested, currentRequested)
	case power > float64(adapter.DefaultConfig.IntermediateSize):
		newRequested = float64(currentDelivered) + power - float64(adapter.DefaultConfig.IntermediateSize)
		log.Log.Infof("IN  found new requested: %f, current requested: %f, current delivered: %f,  power: %f",
			newRequested, currentRequested, currentDelivered, power)

	default:
		log.Log.Infof("Range %d, new requested: %f, current requested: %f, current delivered: %f, power: %f",
			adapter.DefaultConfig.IntermediateSize, newRequested, currentRequested, currentDelivered, power)
	}
	if newRequested > float64(adapter.DefaultConfig.UpperBatLimit) {
		newRequested = float64(adapter.DefaultConfig.UpperBatLimit)
	}
	log.Log.Debugf("Checking limits new requested: %f, current requested: %f, power: %f",
		newRequested, currentRequested, power)
	if newRequested < float64(adapter.DefaultConfig.BaseRequest) {
		newRequested = float64(adapter.DefaultConfig.BaseRequest)
	}
	log.Log.Infof("Power: %f, out: %f, new requested: %f, current requested: %f, current requested: %f",
		power, out, newRequested, currentRequested, currentDelivered)
	p := &paho.Publish{Topic: "energymonitor/update", Payload: []byte(fmt.Sprintf("{\"status\":{\"power\": %f, \"out\": %f, \"requested\": %f}}", power, out, newRequested))}
	ctx := context.Background()
	topic.pahoClient.Publish(ctx, p)

	if newRequested > 0 && newRequested != float64(currentRequested) {
		blockRequestTime = time.Now().Add(time.Duration(adapter.DefaultConfig.WaitAfterRequestSeconds) * time.Second)
		services.ServerMessage("Realtime power request:   %0.1f in [%04d:%04d] power = %0.1f out = %0.1f",
			newRequested, adapter.DefaultConfig.BaseRequest, adapter.DefaultConfig.UpperBatLimit,
			power, out)
		SetOverallPowerConsumption(newRequested)
		refreshCurrentPowerRequest()
	}
}

func SendMqttMessage(topic, message string) {
	p := &paho.Publish{Topic: topic, QoS: 0, Payload: []byte(message)}
	token, err := pahoClient.Publish(context.Background(), p)
	if err != nil {
		log.Log.Errorf("Error sending message: %v", err)
	} else {
		log.Log.Infof("Send MQTT Response: %v", token.ReasonCode)
	}
}
