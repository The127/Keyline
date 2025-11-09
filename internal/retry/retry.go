package retry

import (
	"fmt"
	"time"
)

func FiveTimes(f func() error, msg string) {
	var err error
	for i := 0; i < 5; i++ {
		err = f()
		if err == nil {
			return
		}

		fmt.Printf(msg+": %v\n", err)
		fmt.Printf("Retrying in 5 seconds (attempt %d/5)", i+1)
		time.Sleep(5 * time.Second)
	}

	panic(err)
}
