package voices

import (
	"context"
	"testing"
)

func TestMacSayVoices(t *testing.T) {
	sayVoices := MacSayVoices()
	voices, err := sayVoices.Voices()
	if err != nil {
		t.Fatal(err)
	}

	for _, voice := range voices {
		t.Log(voice)
		err = voice.Say(context.Background(), voice.Detail())
		if err != nil {
			t.Fatal(err)
		}
	}
}
