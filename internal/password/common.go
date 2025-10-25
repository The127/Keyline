package password

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed password-list.txt
var commonPasswordFileContent string

var wordsSet map[string]struct{}

func init() {
	wordsSet = make(map[string]struct{})
	lines := strings.Split(commonPasswordFileContent, "\n")
	for _, line := range lines {
		if line != "" {
			wordsSet[line] = struct{}{}
		}
	}
}

type commonPolicy struct {
}

func (p *commonPolicy) Validate(password string) error {
	if _, ok := wordsSet[password]; ok {
		return fmt.Errorf("password is a common password")
	}
	return nil
}
