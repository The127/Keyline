package clock

import "time"

type Service interface {
	Now() time.Time
}

type TimeSetterFn func(time.Time)

type mockService struct {
	now time.Time
}

func (m *mockService) Now() time.Time {
	return m.now
}

func NewMockServiceNow() (Service, TimeSetterFn) {
	return NewMockService(time.Now())
}

func NewMockService(now time.Time) (Service, TimeSetterFn) {
	service := mockService{
		now: now,
	}
	return &service, func(t time.Time) {
		service.now = t
	}
}

type clockService struct{}

func NewClockService() Service {
	return &clockService{}
}

func (c *clockService) Now() time.Time {
	return time.Now()
}
