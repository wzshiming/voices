package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/wzshiming/voices"
)

var (
	list  = []voices.Voices{voices.BingSayVoices(), voices.MacSayVoices()}
	voice string
	out   string
	files []string
)

func init() {
	flag.StringVarP(&voice, "voice", "v", voice, "voice")
	flag.StringVarP(&out, "out", "o", out, "out")
	flag.StringSliceVarP(&files, "file", "f", files, "file")
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

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
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
		log.Println("not found")
		return
	}

	text := []byte{}
	args := flag.Args()
	if len(args) != 0 {
		text = []byte(strings.Join(args, " "))
		text = append(text, '\n')
	}
	for _, file := range files {
		body, err := openFile(file)
		if err != nil {
			log.Println(err)
			continue
		}
		text = append(text, body...)
		text = append(text, '\n')
	}
	text = bytes.TrimSpace(text)
	if len(text) == 0 {
		return
	}

	if out == "" {
		err := voiceSay.Say(ctx, string(text))
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		err := voiceSay.SayToFile(ctx, out, string(text))
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func openFile(file string) ([]byte, error) {
	if file == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
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
