package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"
	"github.com/wzshiming/voices"
)

var (
	list  = []voices.Voices{voices.BingSayVoices(), voices.MacSayVoices()}
	voice string
	out   string
	file  string
)

func init() {
	flag.StringVarP(&voice, "voice", "v", voice, "voice")
	flag.StringVarP(&out, "out", "o", out, "out")
	flag.StringVarP(&file, "file", "f", file, "file")
	flag.Usage = func() {
		w := os.Stderr
		fmt.Fprintf(w, "Voices:\n")
		helpful()
		fmt.Fprintf(w, "    %s [Options] {text}\n", os.Args[0])
		fmt.Fprintf(w, "    %s -voice Xiaoxiao Hello world\n", os.Args[0])
		fmt.Fprintf(w, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
}

func main() {
	if voice == "" {
		flag.Usage()
		return
	}

	ctx := context.Background()
	var voiceSay voices.Voice
	for _, item := range list {
		vs, err := item.Voices(voices.VoiceName(voice))
		if err != nil {
			log.Println(err)
			continue
		}
		if len(vs) != 0 {
			voiceSay = vs[0]
			break
		}
	}
	if voiceSay == nil {
		log.Printf("not found %q", voice)
		return
	}

	if out != "" {
		save(ctx, voiceSay, file, out)
	} else {
		say(ctx, voiceSay, file)
	}
}

func say(ctx context.Context, voiceSay voices.Voice, file string) {
	content, err := openFile(file)
	if err != nil {
		log.Println(err)
		return
	}

	saysCh := make(chan string, 1)
	cacheCh := make(chan string, 1)

	go func() {
		defer close(saysCh)
		for text := range cacheCh {
			if text != "" {
				_, err := voiceSay.Cache(ctx, text)
				if err != nil {
					log.Println(err)
				}
			}
			saysCh <- text
		}
	}()

	// Read content
	go func() {
		defer func() {
			content.Close()
			close(cacheCh)
		}()
		reader := bufio.NewReader(content)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				if err != io.EOF {
					log.Println(err)
				}
				return
			}
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				cacheCh <- ""
			} else {
				cacheCh <- string(line)
			}
		}
	}()

	index := 0
	for text := range saysCh {
		index++
		log.Printf("%d. %s", index, text)
		if text != "" {
			err := voiceSay.Say(ctx, text)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func save(ctx context.Context, voiceSay voices.Voice, file string, out string) {
	content, err := openFile(file)
	if err != nil {
		log.Println(err)
		return
	}

	text, err := io.ReadAll(content)
	if err != nil {
		content.Close()
		log.Println(err)
		return
	}
	content.Close()

	text = bytes.TrimSpace(text)
	if len(text) == 0 {
		return
	}

	f, err := voiceSay.Cache(ctx, string(text))
	if err != nil {
		log.Println(err)
		return
	}
	err = os.Link(f, out)
	if err != nil {
		log.Println(err)
		return
	}
}

func openFile(file string) (io.ReadCloser, error) {
	if file == "" {
		text := []byte{}
		args := flag.Args()
		if len(args) != 0 {
			text = []byte(strings.Join(args, " "))
			text = append(text, '\n')
		}
		return io.NopCloser(bytes.NewBuffer(text)), nil
	}
	if file == "-" {
		return os.Stdin, nil
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func helpful() {
	ss := []string{}
	for _, item := range list {
		voices, err := item.Voices()
		if err != nil {
			log.Println(err)
			continue
		}
		for _, voice := range voices {
			s := fmt.Sprintf("%s - %s:   \t%s", voice.Language(), voice.Name(), voice.Detail())
			ss = append(ss, s)
		}
	}
	sort.Strings(ss)
	for _, s := range ss {
		fmt.Println(s)
	}
}
