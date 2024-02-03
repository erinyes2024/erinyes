package parser

import (
	"erinyes/logs"
	"sync"
)

var wgParser = sync.WaitGroup{}
var wgInserter = sync.WaitGroup{}

func FileLogParse(repeat bool, sysdigFilepath string, netFilepath string) {
	pChan := make(chan ParsedLog, 1000)
	inserter := Inserter{ParsedLogCh: &pChan}

	concurrencyNum := 10
	for idx := 0; idx < concurrencyNum; idx++ {
		wgInserter.Add(1)
		idx := idx
		go func() {
			defer wgInserter.Done()
			inserter.Insert(idx, repeat)
		}()
	}
	if sysdigFilepath != "" {
		addFileLogParse(NewSysdigParser(&Pusher{&pChan}), sysdigFilepath)
	}
	if netFilepath != "" {
		addFileLogParse(NewNetParser(&Pusher{&pChan}), netFilepath)
	}
	wgParser.Wait()
	close(pChan)
	wgInserter.Wait()
}

func addFileLogParse(_parser Parser, filename string) {
	wgParser.Add(1)
	go func() {
		defer wgParser.Done()
		parser := _parser
		err := ParseFile(filename, parser)
		if err != nil {
			logs.Logger.WithError(err).Fatalf("Parse %s failed", filename)
		}
	}()
}

func HTTPLogParse(repeat bool) {
	pChan := make(chan ParsedLog, 1000)
	inserter := Inserter{ParsedLogCh: &pChan}
	concurrencyNum := 10
	for idx := 0; idx < concurrencyNum; idx++ {
		wgInserter.Add(1)
		idx := idx
		go func() {
			defer wgInserter.Done()
			inserter.Insert(idx, repeat)
		}()
	}

	addHTTPLogParse(NewSysdigParser(&Pusher{&pChan}))
	addHTTPLogParse(NewNetParser(&Pusher{&pChan}))
	wgParser.Wait()
	close(pChan)
	wgInserter.Wait()
}

func addHTTPLogParse(_parser Parser) {
	wgParser.Add(1)
	go func() {
		defer wgParser.Done()
		parser := _parser
		if parser.ParserType() == SYSDIG {
			ParseSysdigChan(parser)
		} else if parser.ParserType() == NET {
			ParseNetChan(parser)
		} else {
			logs.Logger.Errorf("Unknown parser type: %s", parser.ParserType())
		}
	}()
}
