package voices

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	ua                 = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36 Edg/87.0.664.66`
	origin             = `"chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold"`
	trustedClientToken = `6A5AA1D4EAFF4E9FB37E23D68491D6F4`
	listUrl            = `https://speech.platform.bing.com/consumer/speech/synthesize/readaloud/voices/list?trustedclienttoken=` + trustedClientToken
	speechUrl          = `wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1?TrustedClientToken=` + trustedClientToken
	voiceFormat        = "audio-24khz-48kbitrate-mono-mp3"
)

var (
	header = http.Header{
		"Origin":     {origin},
		"User-Agent": {ua},
	}
)

var bing *bingSayVoices

func BingSayVoices() Voices {
	if bing == nil {
		bing = &bingSayVoices{}
	}
	return bing
}

type bingSayVoices struct {
	voices []Voice
}

func (m *bingSayVoices) Voices(opts ...Opt) ([]Voice, error) {
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

func (m *bingSayVoices) getVoices() ([]Voice, error) {
	if m.voices != nil {
		return m.voices, nil
	}

	req, err := http.NewRequest(http.MethodGet, listUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header = header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	list := []bingSayItem{}

	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		return nil, err
	}
	var voices []Voice
	for _, item := range list {
		t := strings.Split(item.ShortName, "-")
		name := t[len(t)-1]
		name = strings.TrimSuffix(name, "Neural")
		voice := bingSay{
			name:        name,
			language:    strings.ReplaceAll(item.Locale, "_", "-"),
			bingSayItem: item,
		}
		voices = append(voices, &voice)
	}
	m.voices = voices
	return voices, nil
}

type bingSay struct {
	name     string
	language string
	bingSayItem
}

func (m bingSay) Name() string {
	return m.name
}

func (m bingSay) Language() string {
	return m.language
}

func (m bingSay) Detail() string {
	return m.bingSayItem.FriendlyName
}

func (m *bingSay) sayReader(ctx context.Context, word string) (io.ReadCloser, error) {
	dl := websocket.Dialer{
		EnableCompression: true,
	}

	conn, resp, err := dl.DialContext(ctx, speechUrl, header)
	if err != nil {
		if resp == nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", err, resp.Status)
	}

	r, w := io.Pipe()

	go func() {
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				w.CloseWithError(err)
				return
			}

			if messageType == websocket.BinaryMessage {
				index := strings.Index(string(p), "Path:audio")
				data := []byte(string(p)[index+12:])
				_, err := w.Write(data)
				if err != nil {
					w.CloseWithError(err)
					return
				}
			} else if messageType == websocket.TextMessage && string(p)[len(string(p))-14:len(string(p))-6] == "turn.end" {
				w.Close()
				return
			}
		}
	}()

	ssml := buildSSML(word, m.bingSayItem.Name)

	t := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	uuid := uuid.NewString()
	cfgMsg := "X-Timestamp:" + t + "\r\nContent-Type:application/json; charset=utf-8\r\n" + "Path:speech.config\r\n\r\n" +
		`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":"false","wordBoundaryEnabled":"false"},"outputFormat":"` + voiceFormat + `"}}}}`

	err = conn.WriteMessage(websocket.TextMessage, []byte(cfgMsg))
	if err != nil {
		return nil, err
	}
	msg := "Path: ssml\r\nX-RequestId: " + uuid + "\r\nX-Timestamp: " + t + "\r\nContent-Type: application/ssml+xml\r\n\r\n" + ssml

	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (m bingSay) cache(ctx context.Context, word string) (string, error) {
	word = clean(word)
	file := filepath.Join(cacheDir, "bing", m.Name(), hashName(word)+".mp3")
	os.MkdirAll(filepath.Dir(file), 0755)
	info, err := os.Stat(file)
	if err == nil && info.Size() != 0 {
		return file, nil
	}

	r, err := m.sayReader(ctx, word)
	if err != nil {
		return "", err
	}
	defer r.Close()

	tmp := file + ".tmp"

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer func() {
		f.Close()
		os.Remove(tmp)
	}()

	n, err := io.Copy(f, r)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("can't read the respone body")
	}

	err = ToMp3(ctx, tmp, file)
	if err != nil {
		return "", err
	}

	return file, nil
}

func (m bingSay) Cache(ctx context.Context, word string) (string, error) {
	return m.cache(ctx, word)
}

func (m bingSay) Say(ctx context.Context, word string) error {
	f, err := m.cache(ctx, word)
	if err != nil {
		return err
	}
	return PlayMp3(ctx, f)
}

func (m bingSay) String() string {
	return m.bingSayItem.FriendlyName
}

type bingSayItem struct {
	Name           string
	ShortName      string
	Gender         string
	Locale         string
	SuggestedCodec string
	FriendlyName   string
	Status         string
}

const ssmlTemplate = `
<speak xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="http://www.w3.org/2001/mstts" xmlns:emo="http://www.w3.org/2009/10/emotionml" version="1.0" xml:lang="en-US">
    <voice name="{voiceName}">
      	<prosody rate="0%" pitch="0%">
			{text}
      	</prosody >
    </voice >
</speak >`

func buildSSML(text, voiceName string) string {
	text = html.EscapeString(text)

	r := strings.ReplaceAll(ssmlTemplate, "{text}", text)
	r = strings.ReplaceAll(r, "{voiceName}", voiceName)
	return r
}
