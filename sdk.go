package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/matoous/go-nanoid/v2"
	"github.com/q-assistant/sdk/config"
	"github.com/q-assistant/sdk/discovery"
	"github.com/q-assistant/sdk/event"
	"github.com/q-assistant/sdk/express"
	"github.com/q-assistant/sdk/logger"
	"github.com/q-assistant/sdk/update"
	"os"
	"os/signal"
	"syscall"
)

type Skill struct {
	id                    string
	vendor                string
	name                  string
	version               string
	discovery             discovery.Discovery
	logger                *logger.Logger
	config                config.Config
	ctx                   context.Context
	cancel                context.CancelFunc
	updates               chan *update.Update
	onConfigUpdateFunc    update.UpdateFunc
	events                *event.Events
	handlers              map[string]HandlerFunc
	dependencies          []string
	express               *express.Express
	allDependenciesOnline chan bool
}

type Data struct {
	Vendor                   string           `json:"vendor"`
	Skill                    string           `json:"skill"`
	Command                  string           `json:"command"`
	Query                    string           `json:"query"`
	Text                     string           `json:"text"`
	AllRequiredParamsPresent bool             `json:"all_required_params_present"`
	Parameters               *structpb.Struct `json:"parameters"`
	OutputContexts           []*Context       `json:"output_contexts"`
}

type Context struct {
	Name     string `json:"name"`
	Lifespan int32  `json:"lifespan"`
}

type HandlerFunc func(logger *logger.Logger, data *Data, express *express.Express)

func NewSkill(vendor string, name string, version string) (*Skill, error) {
	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan *update.Update)

	lgr := logger.NewLogger(name)

	disc, err := discovery.NewConsulClient(ctx, lgr, updates)
	if err != nil {
		cancel()
		return nil, err
	}

	id, _ := gonanoid.New()
	s := &Skill{
		id:           id,
		vendor:       vendor,
		name:         name,
		version:      version,
		ctx:          ctx,
		cancel:       cancel,
		discovery:    disc,
		updates:      updates,
		logger:       lgr,
		handlers:     make(map[string]HandlerFunc),
		events:       event.NewEvents(id, updates),
		dependencies: []string{"expression"},
	}

	s.discovery.WithDependencies(s.dependencies)

	go s.handleUpdates()

	return s, nil
}

// WithConfig allows to set/get config values.
func (s *Skill) WithConfig(data map[string]interface{}) (config.Config, error) {
	var err error

	prefix := fmt.Sprintf("skill/%s/%s", s.vendor, s.name)
	cnf := map[string]interface{}{
		prefix: data,
	}
	if s.config, err = config.NewConsulClient(s.ctx, s.logger, s.updates, prefix, cnf); err != nil {
		return nil, err
	}

	return s.config, nil
}

// OnConfigUpdate is a callback to handle config updates
func (s *Skill) OnConfigUpdate(fn update.UpdateFunc) {
	s.onConfigUpdateFunc = fn
}

func (s *Skill) AddHandler(on string, fn HandlerFunc) {
	topic := fmt.Sprintf("skill/%s/%s/%s", s.vendor, s.name, on)

	s.handlers[topic] = fn
	s.events.Subscribe(topic)
}

func (s *Skill) Run() {
	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	if s.dependencies != nil {
		s.logger.Info("waiting for all dependencies to be online")
		s.allDependenciesOnline = s.discovery.AllDependenciesOnline()

		<-s.allDependenciesOnline
		s.logger.Info("all dependencies are online")
	}

	if err := s.discovery.Register(&discovery.Registration{
		Name: s.name,
	}); err != nil {
		s.logger.Fatal(err)
	}

	s.logger.Info(fmt.Sprintf("%s skill running", s.name))
	<-stop

	s.cancel()
	close(s.updates)
}

func (s *Skill) handleUpdates() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case u := <-s.updates:
			switch u.Kind {
			case update.UpdateKindConfig:
				if s.onConfigUpdateFunc != nil {
					s.onConfigUpdateFunc(u)
				}
			case update.UpdateKindDependency:
				name := u.Update.(string)
				if name == "expression" {
					s.express, _ = express.New(s.discovery.GetConnection("expression"))
					s.logger.Info("expression client created")
				}
			case update.UpdateKindTrigger:
				message := u.Update.(mqtt.Message)
				if fn, ok := s.handlers[message.Topic()]; ok {
					var data *Data
					if err := json.Unmarshal(message.Payload(), &data); err != nil {
						s.logger.Error(err)
						return
					}

					fn(s.logger, data, s.express)
				}
			}
		}
	}
}
