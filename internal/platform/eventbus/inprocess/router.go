package inprocess

import (
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// subscriber delivers events to its middleware-wrapped handler one at a time, so in-flight work drains under Close.
type subscriber struct {
	topic string
	ch    chan eventbus.Event
	stop  chan struct{}
	chain eventbus.Handler
	bus   *Bus
}

func (s *subscriber) run() {
	defer s.bus.wg.Done()
	for {
		select {
		case <-s.stop:
			return
		case <-s.bus.ctx.Done():
			return
		case ev := <-s.ch:
			s.handle(ev)
		}
	}
}

func (s *subscriber) handle(ev eventbus.Event) {
	ctx := logging.CtxWithTraceID(s.bus.ctx, ev.TraceID)
	ctx = logging.CtxWithLogger(ctx, s.bus.log)
	if err := s.chain(ctx, ev); err != nil {
		s.bus.log.Error("event handler failed", "topic", ev.Topic, "id", ev.ID, "err", err)
	}
}
