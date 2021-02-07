package voices

import (
	"io"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
)

func PlayMp3(r io.Reader) error {
	dec, err := mp3.NewDecoder(r)
	if err != nil {
		return err
	}

	context, err := oto.NewContext(dec.SampleRate(), 2, 2, 32*1024)
	if err != nil {
		return err
	}
	defer context.Close()

	player := context.NewPlayer()
	defer player.Close()
	return nil
}
