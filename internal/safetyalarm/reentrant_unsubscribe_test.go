package safetyalarm

import (
	"context"
	"testing"
	"time"
)

func TestSubscriberCanUnsubscribeDuringAlarmDelivery(t *testing.T) {
	ctx := context.Background()
	alarm := Alarm{
		ID:         "alarm-24",
		TenantID:   "tenant-1",
		CraneID:    "crane-24",
		Kind:       "overload",
		ObservedAt: time.Date(2026, 8, 22, 19, 0, 0, 0, time.UTC),
	}

	t.Run("handler removes its own subscription", func(t *testing.T) {
		registry := NewRegistry()
		service := NewService(registry)
		if err := registry.Subscribe("tenant-1", "self-removing", HandlerFunc(func(context.Context, Alarm) error {
			return registry.Unsubscribe("tenant-1", "self-removing")
		})); err != nil {
			t.Fatal(err)
		}
		result := make(chan error, 1)
		go func() { result <- service.Publish(ctx, alarm) }()
		select {
		case err := <-result:
			if err != nil {
				t.Errorf("publish to self-removing handler: %v", err)
			}
			if count := registry.Count("tenant-1"); count != 0 {
				t.Errorf("subscriptions after self-removal = %d, want 0", count)
			}
		case <-time.After(200 * time.Millisecond):
			t.Error("alarm delivery did not return after handler unsubscribed itself")
		}
	})

	t.Run("ordinary handler receives alarm", func(t *testing.T) {
		registry := NewRegistry()
		service := NewService(registry)
		received := make(chan Alarm, 1)
		if err := registry.Subscribe("tenant-1", "observer", HandlerFunc(func(_ context.Context, value Alarm) error {
			received <- value
			return nil
		})); err != nil {
			t.Fatal(err)
		}
		if err := service.Publish(ctx, alarm); err != nil {
			t.Fatalf("publish to ordinary handler: %v", err)
		}
		if value := <-received; value != alarm {
			t.Errorf("received alarm = %+v, want %+v", value, alarm)
		}
		if count := registry.Count("tenant-1"); count != 1 {
			t.Errorf("ordinary subscriptions = %d, want 1", count)
		}
	})
}
