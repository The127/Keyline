package clock

import "time"

type ClockService interface {
	Now() time.Time
}

type clockService struct{}

func NewClockService() ClockService {
	return &clockService{}
}

func (c *clockService) Now() time.Time {
	return time.Now()
}
