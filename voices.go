package voices

import (
	"context"
)

type Voices interface {
	Voices(opts ...Opt) ([]Voice, error)
}

type Voice interface {
	Name() string
	Language() string
	Detail() string
	Say(ctx context.Context, word string) error
	SayToFile(ctx context.Context, file string, word string) error
}

type Opt func(opt *voicesOpt)

func VoiceName(name string) func(opt *voicesOpt) {
	return func(opt *voicesOpt) {
		opt.Name = name
	}
}

func VoiceLanguage(language string) func(opt *voicesOpt) {
	return func(opt *voicesOpt) {
		opt.Language = language
	}
}

type voicesOpt struct {
	Name     string
	Language string
}

func (o *voicesOpt) parse(opts []Opt) {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(o)
	}
}
