package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/q-assistant/sdk/logger"
	"github.com/q-assistant/sdk/update"
	"log"
	"os"
	"strings"
)

type ConsulClient struct {
	data    map[string]interface{}
	client  *api.Client
	addr    string
	logger  *logger.Logger
	ctx     context.Context
	updates chan *update.Update
	watcher *watch.Plan
	prefix  string
}

func NewConsulClient(ctx context.Context, logger *logger.Logger, updates chan *update.Update, prefix string, data map[string]interface{}) (*ConsulClient, error) {
	cnf := api.DefaultConfig()

	addr := os.Getenv("SERVICE_DISCOVERY_ADDRESS")
	if addr != "" {
		cnf.Address = addr
	}

	client, err := api.NewClient(cnf)
	if err != nil {
		return nil, err
	}

	c := &ConsulClient{
		client:  client,
		addr:    addr,
		logger:  logger,
		ctx:     ctx,
		updates: updates,
		prefix:  prefix,
		data:    make(map[string]interface{}),
	}

	if err := c.initial(data); err != nil {
		return nil, err
	}

	return c, nil
}

func (cc *ConsulClient) Get(key string) interface{} {
	return nil
}

func (cc *ConsulClient) Set(key string, data interface{}) {

}

func (cc *ConsulClient) String(path string) string {
	v := cc.getValue(path)
	if v == nil {
		return ""
	}

	switch v.(type) {
	case bool, int, uint, int8, uint8, int16, uint16, int32, uint64, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case string:
		return fmt.Sprintf("%v", v)
	case map[string]interface{}:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

func (cc *ConsulClient) Int(path string) int {
	v := cc.getValue(path)
	if v == nil {
		return 0
	}

	switch v.(type) {
	case bool, int, uint, int8, uint8, int16, uint16, int32, uint64, int64, float32, float64:
		return v.(int)
	default:
		return 0
	}
}

func (cc *ConsulClient) Float(path string) float64 {
	v := cc.getValue(path)
	if v == nil {
		return 0
	}

	switch v.(type) {
	case bool, int, uint, int8, uint8, int16, uint16, int32, uint64, int64, float32, float64:
		return v.(float64)
	default:
		return 0
	}
}

func (cc *ConsulClient) initial(data map[string]interface{}) error {
	for k, c := range data {
		kv, _, err := cc.client.KV().Get(k, nil)
		if err != nil {
			return err
		}

		if kv == nil {
			b, err := json.Marshal(c)
			if err != nil {
				return err
			}

			// empty, time to add
			if _, err := cc.client.KV().Put(&api.KVPair{
				Key:   k,
				Value: b,
			}, nil); err != nil {
				return err
			}

			cc.data[k] = c
		} else {
			current := make(map[string]interface{})
			if err := json.Unmarshal(kv.Value, &current); err != nil {
				return err
			}

			for kk, vv := range data[k].(map[string]interface{}) {
				// Check for any new keys and add them to the
				// current config.
				if _, ok := current[kk]; !ok {
					current[kk] = vv
				}
			}

			b, err := json.Marshal(current)
			if err != nil {
				return err
			}

			if _, err = cc.client.KV().Put(&api.KVPair{
				Key:   kv.Key,
				Value: b,
			}, nil); err != nil {
				return err
			}

			cc.data[kv.Key] = current
			go cc.watch(kv.Key)
		}
	}
	return nil
}

func (cc *ConsulClient) getValue(path string) interface{} {
	tokens := strings.Split(fmt.Sprintf("%s.%s", cc.prefix, path), ".")

	if len(tokens) == 0 {
		return nil
	}

	root := cc.prefix
	if _, ok := cc.data[root]; !ok {
		return nil
	}

	if len(tokens) == 1 {
		return cc.data[root]
	}

	key := tokens[1]
	if _, ok := cc.data[root].(map[string]interface{})[key]; !ok {
		return nil
	}

	return cc.data[root].(map[string]interface{})[key]
}

func (cc *ConsulClient) watch(key string) {
	wp, err := watch.Parse(map[string]interface{}{"type": "key", "key": key})
	if err != nil {
		log.Fatal(err)
	}

	cc.watcher = wp
	cc.watcher.Handler = func(u uint64, i interface{}) {
		kvPair := i.(*api.KVPair)

		if cc.logger != nil {
			cc.logger.Info(fmt.Sprintf("configuration update: %s", kvPair.Key))
		}

		cnf := make(map[string]interface{})
		if err := json.Unmarshal(kvPair.Value, &cnf); err != nil {
			cc.logger.Error(err)
			return
		}

		cc.data[kvPair.Key] = cnf

		cc.updates <- &update.Update{
			Kind: update.UpdateKindConfig,
		}
	}

	go func() {
		if err := cc.watcher.Run(cc.addr); err != nil {
			log.Fatal(err)
		}

		for {
			select {
			case <-cc.ctx.Done():
				cc.watcher.Stop()
				return
			}
		}
	}()
}
