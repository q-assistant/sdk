package express

import (
	"context"
	"github.com/q-assistant/proto/pb/go"
	"google.golang.org/grpc"
)

type Express struct {
	client pb.ExpressionClient
}

func New(conn *grpc.ClientConn) (*Express, error) {
	return &Express{
		client: pb.NewExpressionClient(conn),
	}, nil
}

func (e *Express) Talk(sentence string) {
	e.client.Speak(context.Background(), &pb.Sentence{
		Value:     sentence,
		Broadcast: false,
		Target:    "",
		Context:   nil,
	})
}

func (e *Express) Notify() {

}
