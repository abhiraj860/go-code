package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type Event struct {
	ID int `json:"id"`
	Service string `json:"service"`
	Level string `json:"level"`
	Message string `json:"message"`
}

const fileName = "events.jsonl"

func main() {
	mode := "all"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	switch mode {
	case "write":
		writeJSONL()
	case "read":
		readJSONL()
	case "filter":
		filterJSONL("error")
	case "stdin":
		readStdin()
	default:
		writeJSONL()
		readJSONL()
		filterJSONL("error")
	}
}

func writeJSONL() {
	f, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	events := []Event{
		{1, "api", "info", "request received"},
		{2, "api", "error", "database timeout"},
		{3, "worker", "info", "job started"},
		{4, "worker", "error", "job failed"},
		{5, "api", "warn", "slow response"},
	}
	enc := json.NewEncoder(f)
	for _, e := range events {
		if err := enc.Encode(e); err != nil {
			log.Fatal(err)	
		}
	}
	fmt.Println("wrote", len(events), "records to", fileName)
}

func readJSONL() {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024 * 1024), 1024 * 1024)	
	count := 0
	for sc.Scan() {
		line := sc.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			log.Println("skipping bad line:", err)
			continue
		}
		count++
		fmt.Printf("%d [%s] %s: %s \n", e.ID, e.Level, e.Service, e.Message) 
	}
	if err := sc.Err(); err != nil {
		log.Fatal("read error:", err)
	}
	fmt.Println("read", count, "records")
}

func filterJSONL(level string) {
	in, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create("filtered.jsonl")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	sc := bufio.NewScanner(in)
	enc := json.NewEncoder(out)
	kept := 0
	for sc.Scan() {
		var e Event 
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		if e.Level != level {
			continue
		}
		if err := enc.Encode(e); err != nil {
			log.Fatal(err)
		}
		kept++
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("Error while scanning %s", err)		
	}
	fmt.Printf("kept %d %q records in filtered.jsonl\n", kept, level)
}

func readStdin() {
	sc := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for sc.Scan() {
		var e Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		e.Level = strings.ToUpper(e.Level)
		enc.Encode(e)
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("Error while scanning %s", err)		
	}
}






















