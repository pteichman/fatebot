package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"unicode"

	"bitbucket.org/tebeka/snowball"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/pteichman/fate"
)

type Config struct {
	Server      string    `json:"server"`
	Password    string    `json:"password"`
	SSL         bool      `json:"ssl"`
	Nick        string    `json:"nick"`
	IgnoreNicks []string  `json:"ignore_nicks"`
	Channels    []Channel `json:"channels"`
}

type Channel struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

var (
	configFile = flag.String("config", "", "config file")
	pprof      = flag.String("pprof", "", "run http server (host:port)")
)

func main() {
	flag.Parse()

	var config Config
	if *configFile != "" {
		configBytes, err := ioutil.ReadFile(*configFile)
		if err != nil {
			log.Fatal("Reading configFile: " + err.Error())
		}

		err = json.Unmarshal(configBytes, &config)
		if err != nil {
			log.Fatal("Unmarshaling configFile: " + err.Error())
		}
	}

	if *pprof != "" {
		go func() {
			log.Fatal(http.ListenAndServe(*pprof, nil))
		}()
	}

	model := fate.NewModel(fate.Config{Stemmer: newStemmer()})

	for _, f := range flag.Args() {
		err := learnFile(model, f)
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
	}

	RunForever(model, config)
}

type stemmer struct {
	tran     transform.Transformer
	snowball *snowball.Stemmer
}

func newStemmer() stemmer {
	isRemovable := func(r rune) bool {
		return unicode.Is(unicode.Mn, r) || unicode.IsPunct(r)
	}

	stem, _ := snowball.New("english")

	return stemmer{
		tran:     transform.Chain(norm.NFD, transform.RemoveFunc(isRemovable), norm.NFC),
		snowball: stem,
	}
}

func (s stemmer) Stem(word string) string {
	str, _, _ := transform.String(s.tran, word)
	return Squish(s.snowball.Stem(strings.ToLower(str)), 2)
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
