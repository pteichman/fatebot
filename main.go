package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"strings"
	"unicode"

	"bitbucket.org/tebeka/snowball"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/pteichman/fate"
)

var (
	ircserver  = flag.String("irc.server", "", "irc server (host:port)")
	ircchannel = flag.String("irc.channels", "#fate", "irc channels")
	ircnick    = flag.String("irc.nick", "fate", "irc nickname")
)

func main() {
	flag.Parse()

	config := fate.Config{Stemmer: newStemmer()}
	model := fate.NewModel(config)

	for _, f := range flag.Args() {
		err := learnFile(model, f)
		if err != nil {
			log.Fatal("Error: %s\n", err)
		}
	}

	opts := &Options{
		Server:   *ircserver,
		Nick:     *ircnick,
		Channels: []string{*ircchannel},
	}

	RunForever(model, opts)
}

type stemmer struct {
	tran transform.Transformer
	snowball *snowball.Stemmer
}

func newStemmer() stemmer {
	isRemovable := func(r rune) bool {
		return unicode.Is(unicode.Mn, r) || unicode.IsPunct(r)
	}

	stem, _ := snowball.New("english")

	return stemmer{
		tran: transform.Chain(norm.NFD, transform.RemoveFunc(isRemovable), norm.NFC),
		snowball: stem,
	}
}

func (s stemmer) Stem(word string) string {
	str, _, _ := transform.String(s.tran, word)
	return s.snowball.Stem(strings.ToLower(str))
}

func learnFile(m *fate.Model, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	s := bufio.NewScanner(bufio.NewReader(f))
	for s.Scan() {
		m.Learn(s.Text())
	}

	return s.Err()
}
