package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

type SplitToken struct {
	id     string
	secret string
}

var (
	ErrDecodingToken = errors.New("error decoding token")
)

func DecodeSplitToken(base64Token string) (SplitToken, error) {
	decodedBytes, err := base64.RawURLEncoding.DecodeString(base64Token)
	if err != nil {
		return SplitToken{}, fmt.Errorf("base64 decoding token: %w", err)
	}

	decodedString := string(decodedBytes)
	tokenParts := strings.Split(decodedString, ":")
	if len(tokenParts) != 2 {
		return SplitToken{}, fmt.Errorf("token has not exactly two parts: %w", ErrDecodingToken)
	}

	token := NewSplitToken(tokenParts[0], tokenParts[1])
	if token.id == "" {
		return SplitToken{}, fmt.Errorf("missing id: %w", ErrDecodingToken)
	}
	if token.secret == "" {
		return SplitToken{}, fmt.Errorf("missing secret: %w", ErrDecodingToken)
	}

	return token, nil
}

func NewSplitToken(id string, secret string) SplitToken {
	return SplitToken{
		id:     id,
		secret: secret,
	}
}

func (t *SplitToken) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", t.id, t.secret)))
}

func (t *SplitToken) Id() string {
	return t.id
}

func (t *SplitToken) Secret() string {
	return t.secret
}
