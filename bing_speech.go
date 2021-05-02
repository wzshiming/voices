package voices

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wzshiming/requests"
	"golang.org/x/net/websocket"
)

const (
	ua                 = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36 Edg/87.0.664.66`
	trustedClientToken = `6A5AA1D4EAFF4E9FB37E23D68491D6F4`
	listUrl            = `https://speech.platform.bing.com/consumer/speech/synthesize/readaloud/voices/list?trustedclienttoken=` + trustedClientToken
	speechUrl          = `wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1?TrustedClientToken=` + trustedClientToken
)

var bing *bingSayVoices

func BingSayVoices() Voices {
	if bing == nil {
		bing = &bingSayVoices{
			req: requests.NewClient().
				SetLogLevel(requests.LogIgnore).
				SetCache(requests.FileCacheDir(cacheDir)).
				NewRequest().
				SetUserAgent(ua),
		}
	}
	return bing
}

type bingSayVoices struct {
	req    *requests.Request
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
	resp, err := m.req.Get(listUrl)
	if err != nil {
		return nil, err
	}
	list := []bingSayItem{}
	err = json.Unmarshal(resp.Body(), &list)
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
	conn, err := websocket.Dial(speechUrl, "", "https://www.bing.com/")
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	const head = "Content-Type:application/json; charset=utf-8\r\n\r\nPath:speech.config\r\n\r\n{\"context\":{\"synthesis\":{\"audio\":{\"metadataoptions\":{\"sentenceBoundaryEnabled\":\"false\",\"wordBoundaryEnabled\":\"true\"},\"outputFormat\":\"audio-24khz-160kbitrate-mono-mp3\"}}}}\r\n"
	var body = "X-RequestId:fe83fbefb15c7739fe674d9f3e81d38f\r\nContent-Type:application/ssml+xml\r\nPath:ssml\r\n\r\n<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='en-US'><voice  name='" + m.bingSayItem.Name + "'><prosody pitch='+0Hz' rate ='+0%' volume='+0%'>" + html.EscapeString(word) + "</prosody></voice></speak>\r\n"
	_, err = conn.Write([]byte(head))
	if err != nil {
		return nil, err
	}
	_, err = conn.Write([]byte(body))
	if err != nil {
		return nil, err
	}

	return struct {
		io.Reader
		io.Closer
	}{
		Reader: newBindStream(ctx, conn),
		Closer: conn,
	}, nil
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

type bingStream struct {
	ctx context.Context
	r   io.Reader
}

func newBindStream(ctx context.Context, r io.Reader) io.Reader {
	return &bingStream{
		ctx: ctx,
		r:   r,
	}
}

func (b *bingStream) Read(p []byte) (n int, err error) {
	err = b.ctx.Err()
	if err != nil {
		return 0, err
	}
	n, err = b.r.Read(p)
	if err != nil {
		return 0, err
	}
	if n > 2 && p[0] == 0 {
		o := p[:n]
		if o[1] == 128 {
			if bytes.Contains(o, []byte("Content-Type:audio/mpeg\r\n")) {
				sl := []byte("Path:audio\r\n")
				i := bytes.Index(o, sl)
				tmp := o[i+len(sl) : n]
				n = copy(o, tmp)
				return n, nil
			}
		} else if o[1] == 103 {
			return 0, io.EOF
		}
	}
	return b.Read(p)
}
