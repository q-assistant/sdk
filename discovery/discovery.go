package discovery

import (
	"fmt"
	"google.golang.org/grpc"
)

const (
	HealthPassing  = "passing"
	HealthWarning  = "warning"
	HealthCritical = "critical"
	HealthMaint    = "maintenance"
)

type Discovery interface {
	// WithDependencies When dependencies are provided, they should be checked
	WithDependencies(dependencies []string)

	// AllDependenciesOnline returns true when all are online
	AllDependenciesOnline() chan bool

	// Register a service to be discovered
	Register(registration *Registration) error

	// DeRegister a service, this will make the service un-discoverable
	DeRegister() error

	// Find a service by it's name
	Find(name string) ([]*Service, error)

	// GetConnection returns a connection to a service
	GetConnection(name string) *grpc.ClientConn
}

type Registration struct {
	Port    int
	Name    string
	Address string
	Tags    []string
	Meta    map[string]string
}

type Service struct {
	Port    int
	Name    string
	Address string
	Meta    map[string]string
	Id      string
}

func (s *Service) BuildServiceClient() (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", s.Address, s.Port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return conn, nil
}
