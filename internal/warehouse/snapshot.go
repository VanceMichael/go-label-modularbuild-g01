package warehouse

import (
	"sort"

	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type CalendarSnapshot struct {
	Slots    []Slot
	Bookings []Booking
}

func (c *Calendar) Export() CalendarSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	snapshot := CalendarSnapshot{
		Slots:    make([]Slot, 0, len(c.slots)),
		Bookings: make([]Booking, 0, len(c.bookings)),
	}
	for _, slot := range c.slots {
		snapshot.Slots = append(snapshot.Slots, slot)
	}
	for _, booking := range c.bookings {
		snapshot.Bookings = append(snapshot.Bookings, booking)
	}
	sort.Slice(snapshot.Slots, func(i, j int) bool { return snapshot.Slots[i].ID < snapshot.Slots[j].ID })
	sort.Slice(snapshot.Bookings, func(i, j int) bool { return snapshot.Bookings[i].ID < snapshot.Bookings[j].ID })
	return snapshot
}

func Restore(snapshot CalendarSnapshot) (*Calendar, error) {
	calendar := &Calendar{slots: make(map[string]Slot, len(snapshot.Slots))}
	for _, slot := range snapshot.Slots {
		if slot.ID == "" || slot.Terminal == "" || slot.MaxKg <= 0 || slot.UsedKg < 0 ||
			slot.UsedKg > slot.MaxKg || !slot.CloseAt.After(slot.OpenAt) {
			return nil, domain.ErrInvalid
		}
		if _, exists := calendar.slots[slot.ID]; exists {
			return nil, domain.ErrConflict
		}
		calendar.slots[slot.ID] = slot
	}
	for _, booking := range snapshot.Bookings {
		if booking.ID == "" || booking.ModuleMoveID == "" || booking.SlotID == "" || booking.WeightKg <= 0 ||
			!booking.End.After(booking.Start) {
			return nil, domain.ErrInvalid
		}
		if _, exists := calendar.slots[booking.SlotID]; !exists {
			return nil, domain.ErrNotFound
		}
		if _, exists := calendar.bookings[booking.ID]; exists {
			return nil, domain.ErrConflict
		}
		calendar.bookings[booking.ID] = booking
	}
	return calendar, nil
}
