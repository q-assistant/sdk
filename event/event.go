package event

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/q-assistant/sdk/update"
	"time"
)

type Events struct {
	handlers map[string]HandlerFunc
	updates  chan *update.Update
	client   mqtt.Client
}

func NewEvents(id string, updates chan *update.Update) *Events {
	opts := mqtt.NewClientOptions()
	opts.SetClientID(id)
	opts.AddBroker("tcp://localhost:1883")
	opts.SetOrderMatters(false)       // Allow out of order messages (use this option unless in order delivery is essential)
	opts.ConnectTimeout = time.Second // Minimal delays on connect
	opts.WriteTimeout = time.Second   // Minimal delays on writes
	opts.KeepAlive = 10               // Keepalive every 10 seconds so we quickly detect network outages
	opts.PingTimeout = time.Second    // local broker so response should be quick

	// Automate connection management (will keep trying to connect and will reconnect if network drops)
	opts.ConnectRetry = true
	opts.AutoReconnect = true

	opts.OnConnectionLost = func(cl mqtt.Client, err error) {
		fmt.Println("connection lost")
	}
	opts.OnConnect = func(mqtt.Client) {
		fmt.Println("connection established")
	}
	opts.OnReconnecting = func(mqtt.Client, *mqtt.ClientOptions) {
		fmt.Println("attempting to reconnect")
	}

	client := mqtt.NewClient(opts)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return &Events{
		client: client,
		updates: updates,
	}
}

func (e *Events) Subscribe(topic string) {
	token := e.client.Subscribe(topic, 1, func(client mqtt.Client, message mqtt.Message) {
		e.updates <- &update.Update{
			Kind:   update.UpdateKindTrigger,
			Update: message,
		}

		message.Ack()
	})

	token.Wait()
}
