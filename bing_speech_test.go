package voices

import (
	"context"
	"testing"
)

func TestBingSayVoices(t *testing.T) {
	sayVoices := BingSayVoices()
	voices, err := sayVoices.Voices()
	if err != nil {
		t.Fatal(err)
	}

	for _, voice := range voices {
		t.Log(voice)
		err = voice.Say(context.Background(), "臭猪臭猪")
		if err != nil {
			t.Fatal(err)
		}
	}
}
