package notification

import (
	"context"
	"fmt"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/workerpool"
	"strings"
	"sync"
	"time"
)

type Message struct {
	ID        string
	TenantID  string
	Recipient string
	Channel   string
	Subject   string
	Body      string
	CreatedAt time.Time
}
type Sender interface {
	Send(context.Context, Message) error
}
type DeliveryResult struct {
	MessageID string
	Error     string
}

type MemorySender struct {
	mu       sync.Mutex
	Sent     []Message
	Failures int
}

func (m *MemorySender) Send(ctx context.Context, v Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Failures > 0 {
		m.Failures--
		return fmt.Errorf("temporary notification failure")
	}
	m.Sent = append(m.Sent, v)
	return nil
}

type Router struct{ senders map[string]Sender }

func New() *Router { return &Router{senders: map[string]Sender{}} }
func (r *Router) Register(channel string, s Sender) error {
	if strings.TrimSpace(channel) == "" || s == nil {
		return domain.ErrInvalid
	}
	if _, ok := r.senders[channel]; ok {
		return domain.ErrConflict
	}
	r.senders[channel] = s
	return nil
}
func (r *Router) Deliver(ctx context.Context, v Message) error {
	if v.ID == "" || v.Recipient == "" || v.Body == "" {
		return domain.ErrInvalid
	}
	s, ok := r.senders[v.Channel]
	if !ok {
		return domain.ErrNotFound
	}
	return s.Send(ctx, v)
}

func (r *Router) DeliverBatch(ctx context.Context, workers int, messages []Message) []DeliveryResult {
	jobs := make([]workerpool.Job, 0, len(messages))
	for _, message := range messages {
		message := message
		jobs = append(jobs, func(jobCtx context.Context) error {
			return r.Deliver(jobCtx, message)
		})
	}
	completed := workerpool.Run(ctx, workers, jobs)
	results := make([]DeliveryResult, 0, len(completed))
	for _, result := range completed {
		item := DeliveryResult{MessageID: messages[result.Index].ID}
		if result.Err != nil {
			item.Error = result.Err.Error()
		}
		results = append(results, item)
	}
	return results
}
