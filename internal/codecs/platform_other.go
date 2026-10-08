//go:build !windows

package codecs

import (
	"errors"
	"io"
)

type platformDecoder struct{}

func (platformDecoder) Decode(io.Reader) (Stream, error) {
	return Stream{}, errors.New("AAC, M4A and WMA currently require Windows Media Foundation")
}
