package clock

import "time"

type Service interface {
	Now() time.Time
}

type clockService struct{}

func NewClockService() Service {
	return &clockService{}
}

func (c *clockService) Now() time.Time {
	return time.Now()
}
