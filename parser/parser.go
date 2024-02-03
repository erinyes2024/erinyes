package parser

import (
	"bufio"
	"erinyes/logs"
	"os"
)

type Parser interface {
	ParsePushLine(rawLine string) error
	ParserType() string
}

const (
	SYSDIG string = "sysdig"
	NET    string = "net"
)

type Pusher struct {
	parsedLogCh *chan ParsedLog
}

func (p *Pusher) PushParsedLog(pl ParsedLog) error {
	*p.parsedLogCh <- pl
	return nil
}

func ParseFile(name string, parser Parser) error {
	f, err := os.Open(name)
	if err != nil {
		logs.Logger.WithError(err).Errorf("Open file %s failed", name)
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(bufio.NewReader(f))

	for s.Scan() {
		line := s.Text()
		err = parser.ParsePushLine(line)
		if err != nil {
			return err
		}
	}

	return nil
}

var SysdigRawChan chan string
var NetRawChan chan string

func ParseSysdigChan(parser Parser) {
	SysdigRawChan = make(chan string, 1000)
	for rawString := range SysdigRawChan {
		err := parser.ParsePushLine(rawString)
		if err != nil {
			logs.Logger.Errorf("parse sysdig log failed: %s", rawString)
		}
	}
}

func ParseNetChan(parser Parser) {
	NetRawChan = make(chan string, 1000)
	for rawString := range NetRawChan {
		err := parser.ParsePushLine(rawString)
		if err != nil {
			logs.Logger.Errorf("parse net log failed: %s", rawString)
		}
	}
}
