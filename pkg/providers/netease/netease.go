// NOTE:not supported
package netease

import (
	"github.com/rancher/receiver/pkg/providers"
)

const (
	Name = "netease"
)

type sender struct {
}

func New(opt map[string]string) (providers.Sender, error) {
	return &sender{}, nil
}

func (s *sender) Send(msg string, receiver providers.Receiver) error {
	return nil
}
