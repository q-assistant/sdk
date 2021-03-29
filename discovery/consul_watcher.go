package discovery

import (
	"fmt"
	"github.com/q-assistant/sdk/update"
	"google.golang.org/grpc"
	"time"
)

// watch keeps dependencies up to date
func (cc *ConsulClient) watch() {
	if len(cc.dependencies) == 0 {
		return
	}

	cc.connections = make(map[string]*grpc.ClientConn)
	go cc.doWatch()
}

func (cc *ConsulClient) doWatch() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-cc.ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			for _, dep := range cc.dependencies {

				services, err := cc.Find(dep)
				if err != nil {
					cc.logger.Error(fmt.Sprintf("failed to find services %s: %s", dep, err))
					continue
				}

				if len(services) == 0 {
					// remove them
					if _, ok := cc.connections[dep]; ok {
						delete(cc.connections, dep)

						cc.updates <- &update.Update{
							Kind: update.UpdateKindDependency,
						}

						cc.logger.Info(fmt.Sprintf("service '%s' disconnected", dep))
					}

					cc.allOnline()
					continue
				}

				// pick first service
				service := services[0]
				if _, ok := cc.connections[dep]; !ok {
					cc.connections[dep], _ = service.BuildServiceClient()

					cc.updates <- &update.Update{
						Kind:   update.UpdateKindDependency,
						Update: dep,
					}

					cc.logger.Info(fmt.Sprintf("service '%s' connected at %s:%d", dep, service.Address, service.Port))
				}

				cc.allOnline()
			}
		}
	}
}

func (cc *ConsulClient) allOnline() {
	if len(cc.connections) == len(cc.dependencies) {
		cc.allDependenciesOnlineChan <- true
	} else {
		cc.allDependenciesOnlineChan <- false
	}
}
