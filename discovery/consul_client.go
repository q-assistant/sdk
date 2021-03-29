package discovery

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/matoous/go-nanoid/v2"
	"github.com/q-assistant/sdk/logger"
	"github.com/q-assistant/sdk/update"
	"google.golang.org/grpc"
	"os"
	"time"
)

type ConsulClient struct {
	id                        string
	client                    *api.Client
	dependencies              []string
	connections               map[string]*grpc.ClientConn
	logger                    *logger.Logger
	allDependenciesOnlineChan chan bool
	ctx                       context.Context
	updates                   chan *update.Update
}

func NewConsulClient(ctx context.Context, logger *logger.Logger, updates chan *update.Update) (*ConsulClient, error) {
	cnf := api.DefaultConfig()

	addr := os.Getenv("SERVICE_DISCOVERY_ADDRESS")
	if addr != "" {
		cnf.Address = addr
	}

	client, err := api.NewClient(cnf)
	if err != nil {
		return nil, err
	}

	return &ConsulClient{
		client:  client,
		logger:  logger,
		ctx:     ctx,
		updates: updates,
	}, nil
}

func (cc *ConsulClient) WithDependencies(dependencies []string) {
	cc.dependencies = dependencies
	cc.allDependenciesOnlineChan = make(chan bool)
	cc.watch()
}

func (cc *ConsulClient) AllDependenciesOnline() chan bool {
	return cc.allDependenciesOnlineChan
}

func (cc *ConsulClient) Register(registration *Registration) error {
	cc.id, _ = gonanoid.New()

	err := cc.client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		Kind: api.ServiceKindTypical,
		ID:   cc.id,
		Name: registration.Name,
		Tags: append([]string{"skill"}, registration.Tags...),
		// Port:    registration.Port,
		// Address: registration.Address,
		Meta: registration.Meta,
		Check: &api.AgentServiceCheck{
			CheckID:                        cc.id,
			TTL:                            "2s",
			DeregisterCriticalServiceAfter: "4s",
		},
	})

	if err != nil {
		return err
	}

	go cc.statusUpdate()

	return nil
}

func (cc *ConsulClient) DeRegister() error {
	return cc.client.Agent().ServiceDeregister(cc.id)
}

func (cc *ConsulClient) Find(name string) ([]*Service, error) {
	results, _, err := cc.client.Health().Service(name, "core", true, nil)
	if err != nil {
		return nil, err
	}

	services := make([]*Service, len(results))
	for i, service := range results {
		services[i] = &Service{
			Port:    service.Service.Port,
			Name:    service.Service.Service,
			Address: service.Service.Address,
		}
	}

	return services, nil
}

func (cc *ConsulClient) GetConnection(name string) *grpc.ClientConn {
	if conn, ok := cc.connections[name]; ok {
		return conn
	}

	return nil
}

func (cc *ConsulClient) setStatus(note string, status string) error {
	if err := cc.client.Agent().UpdateTTL(cc.id, note, status); err != nil {
		return fmt.Errorf("discovery.consul: %w", err)
	}

	return nil
}

func (cc *ConsulClient) statusUpdate() {
	ticker := time.NewTicker(time.Second)

	go func() {
		select {
		case <-cc.ctx.Done():
			ticker.Stop()
		}
	}()

	for range ticker.C {
		if err := cc.setStatus("running", HealthPassing); err != nil {
			ticker.Stop()
		}
	}
}
