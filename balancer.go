package dynproxy

import (
	"context"
	"github.com/rs/zerolog/log"
	"net"
)

const (
	SingleServerStrategy    = 0
	JumpHashStrategy        = 1
	RoundRobinStrategy      = 2
	LeastConnectionStrategy = 3
)

var balancers map[string]*Balancer

type Balancer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	Name          string
	Backends      []*Backend
	updateChannel chan status
}

func InitBalancers(ctx context.Context, config Config) {
	balancers = make(map[string]*Balancer)
	for _, balancerConfig := range config.Backends {
		name := balancerConfig.Name
		backends := make([]*Backend, 0)
		balancerCtx, cancelFunc := context.WithCancel(ctx)
		notifyChannel := make(chan status, 10)
		for _, backendConfig := range balancerConfig.Backends {
			backendCtx := context.WithValue(balancerCtx, "channel", notifyChannel)
			backend := &Backend{
				ctx:     backendCtx,
				Name:    backendConfig.Name,
				Net:     backendConfig.Net,
				Address: backendConfig.Address,
				HealthCheck: &HealthCheck{
					Period: backendConfig.HealthCheckPeriod,
				},
			}
			backend.initBackend()
			backends = append(backends, backend)
		}
		balancer := &Balancer{
			Name:          name,
			ctx:           balancerCtx,
			cancel:        cancelFunc,
			Backends:      backends,
			updateChannel: notifyChannel,
		}
		go balancer.start()
		balancers[name] = balancer
	}
}

func getConnByBalancerName(name string) (net.Conn, error) {
	balancer, ok := balancers[name]
	if !ok {
		return nil, balancerNotFound
	}
	return balancer.getNextBackendConn()
}

func (b *Balancer) getNextBackendConn() (net.Conn, error) {
	if len(b.Backends) > 0 {
		// todo: need to use ipaddress of the frontend connection
		//backend := b.Backends[JumpHash(uint64(time.Now().UnixNano()), len(b.Backends))]
		backend := b.Backends[0]
		conn, err := backend.getBackendConn()
		if err != nil {
			return nil, nil
		}
		setSocketOptions(conn)
		return conn, nil
	}
	return nil, noActiveBackends
}

func (b *Balancer) start() {
	for {
		select {
		case <-b.ctx.Done():
			log.Info().Msgf("stopping balancer:%s", b.Name)
			return
		case state := <-b.updateChannel:
			log.Debug().Msgf("Received %+v", state)
			// TODO: Need to update list of active backends based on the channel updates
		}
	}
}
