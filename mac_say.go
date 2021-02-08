package voices

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var mac *macSayVoices

func MacSayVoices() Voices {
	if mac == nil {
		mac = &macSayVoices{}
	}
	return mac
}

type macSayVoices struct {
	voices []Voice
}

func (m *macSayVoices) Voices(opts ...Opt) ([]Voice, error) {
	vs, err := m.getVoices()
	if err != nil {
		return nil, err
	}
	opt := voicesOpt{}
	opt.parse(opts)
	if opt.Name != "" {
		for _, voice := range vs {
			if voice.Name() == opt.Name {
				return []Voice{voice}, nil
			}
		}
		return nil, fmt.Errorf("not found voice %q", opt.Name)
	} else if opt.Language != "" {
		voices := []Voice{}
		for _, voice := range vs {
			if voice.Language() == opt.Language {
				voices = append(voices, voice)
			}
		}
		return voices, nil
	} else {
		return vs, nil
	}
}

func (m *macSayVoices) getVoices() ([]Voice, error) {
	if m.voices != nil {
		return m.voices, nil
	}
	out, err := exec.Command("say", "-v", "?").Output()
	if err != nil {
		return nil, err
	}
	var voices []Voice
	rows := bytes.Split(out, []byte{'\n'})
	for _, row := range rows {
		row = bytes.TrimSpace(row)
		i := bytes.Index(row, []byte{' '})
		if i == -1 {
			continue
		}
		name := string(bytes.TrimSpace(row[:i]))
		row = bytes.TrimSpace(row[i:])
		i = bytes.Index(row, []byte{' '})
		if i == -1 {
			continue
		}
		language := string(bytes.TrimSpace(row[:i]))
		detail := string(bytes.TrimPrefix(bytes.TrimSpace(row[i:]), []byte("# ")))
		voice := macSay{
			name:     name,
			language: strings.ReplaceAll(language, "_", "-"),
			detail:   detail,
		}
		voices = append(voices, voice)
	}
	m.voices = voices
	return voices, nil
}

type macSay struct {
	name     string
	language string
	detail   string
}

func (m macSay) Name() string {
	return m.name
}

func (m macSay) Language() string {
	return m.language
}

func (m macSay) Detail() string {
	return m.detail
}

func (m macSay) sayToFile(ctx context.Context, word string) (string, error) {
	word = clean(word)
	file := filepath.Join(cacheDir, "mac_say", m.Name(), hashName(word)+".mp3")
	os.MkdirAll(filepath.Dir(file), 0755)
	info, err := os.Stat(file)
	if err == nil && info.Size() != 0 {
		return file, nil
	}

	tmp := file + ".aac"
	err = executive(ctx, "say", "-o", tmp, "-v", m.name, word)
	if err != nil {
		return "", err
	}
	err = ToMp3(ctx, tmp, file)
	if err != nil {
		return "", err
	}
	os.Remove(tmp)
	return file, nil
}

func (m macSay) SayToFile(ctx context.Context, file string, word string) error {
	f, err := m.sayToFile(ctx, word)
	if err != nil {
		return err
	}
	return os.Link(f, file)
}

func (m macSay) Say(ctx context.Context, word string) error {
	f, err := m.sayToFile(ctx, word)
	if err != nil {
		return err
	}
	return PlayMp3(ctx, f)
}

func (m macSay) String() string {
	return fmt.Sprintf("%s(%s): %s", m.name, m.language, m.detail)
}
