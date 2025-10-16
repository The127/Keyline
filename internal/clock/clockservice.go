package clock

import "time"

type Service interface {
	Now() time.Time
}

type mockService struct {
	now time.Time
}

func (m *mockService) Now() time.Time {
	return m.now
}

func NewMockServiceNow() Service {
	return NewMockService(time.Now())
}

func NewMockService(now time.Time) Service {
	return &mockService{
		now: now,
	}
}

type clockService struct{}

func NewClockService() Service {
	return &clockService{}
}

func (c *clockService) Now() time.Time {
	return time.Now()
}
