package gnet

import "github.com/liyee/gray/giface"

type Option func(s *Server)

func WithPacket(pack giface.IDataPack) Option {
	return func(s *Server) {
		s.SetPacket(pack)
	}
}

// Options for Client
type ClientOption func(c giface.IClient)

// Implement custom data packet format by implementing the Packet interface for client,
// otherwise use the default data packet format
func WithPacketClient(pack giface.IDataPack) ClientOption {
	return func(c giface.IClient) {
		c.SetPacket(pack)
	}
}

// Set client name
func WithNameClient(name string) ClientOption {
	return func(c giface.IClient) {
		c.SetName(name)
	}
}
