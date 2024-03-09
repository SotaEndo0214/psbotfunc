package pokemonsleep

import (
	"io"

	"go.uber.org/zap"
)

type Image struct {
	Logger *zap.Logger `json:"-"`
	Format string
	Bytes  io.Reader

	Width  int
	Height int
}

func NewImage(imageBytes io.Reader, filetype string, width, height int, logger *zap.Logger) (*Image, error) {
	return &Image{
		Logger: logger,
		Bytes:  imageBytes,
		Format: filetype,
		Width:  width,
		Height: height,
	}, nil
}
