package dynproxy

type Balancer struct {
	Name     string
	Backends []*Backend
}

func (b *Balancer) GetBackend() *Backend {
	if len(b.Backends) > 0 {
		return b.Backends[0]
	}
	return nil
}
