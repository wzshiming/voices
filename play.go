package voices

import (
	"io"
	"os"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
)

func PlayMp3FromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return PlayMp3(f)
}

func PlayMp3(r io.Reader) error {
	dec, err := mp3.NewDecoder(r)
	if err != nil {
		return err
	}

	context, err := oto.NewContext(dec.SampleRate(), 2, 2, 64*1024)
	if err != nil {
		return err
	}
	defer context.Close()

	player := context.NewPlayer()
	defer player.Close()

	_, err = io.Copy(player, dec)
	if err != nil {
		return err
	}
	return nil
}
