package hystrix

import "github.com/gojek/hystrix-go/hystrix/rolling"

type executorPool struct {
	Name              string
	MaxActiveRequests *rolling.Number
	Executed          *rolling.Number
	Max               int
	Tickets           chan *struct{}
}

func newExecutorPool(name string) *executorPool {
	maxRequests := getSettings(name).MaxConcurrentRequests
	tickets := make(chan *struct{}, maxRequests)
	for range maxRequests {
		tickets <- &struct{}{}
	}

	return &executorPool{
		Name:              name,
		MaxActiveRequests: rolling.NewNumber(),
		Executed:          rolling.NewNumber(),
		Max:               maxRequests,
		Tickets:           tickets,
	}
}

func (p *executorPool) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.Executed.Increment(1)
	p.MaxActiveRequests.UpdateMax(float64(p.ActiveCount()))
	p.Tickets <- ticket
}

func (p *executorPool) ActiveCount() int {
	return p.Max - len(p.Tickets)
}

func (p *executorPool) ResetMetrics() {
	p.MaxActiveRequests.Reset()
	p.Executed.Reset()
}
