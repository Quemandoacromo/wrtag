//go:build none

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/shlex"
)

func main() {
	flag.Parse()
	inPath, outPath := flag.Arg(0), flag.Arg(1)

	inFile, err := os.Open(inPath)
	cerr(err)
	defer inFile.Close()

	tagList := map[string][]string{}

	var reading bool
L:
	for sc := bufio.NewScanner(inFile); sc.Scan(); {
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}

		switch text {
		case "const (":
			reading = true
			continue
		case ")":
			if reading {
				break L
			}
		}
		if !reading {
			continue
		}

		// line format
		const (
			_ = iota
			_
			tag
			comment
			instr
		)

		l, err := shlex.Split(text)
		cerr(err)

		tagList[l[tag]] = nil
		if len(l)-1 >= comment && l[comment] == "//tag:" && l[instr] == "alts" {
			tagList[l[tag]] = append(tagList[l[tag]], l[instr+1:]...)
		}
	}

	var tagKeys []string
	for k := range tagList {
		tagKeys = append(tagKeys, k)
	}
	sort.Strings(tagKeys)

	outFile, err := os.Create(outPath)
	cerr(err)
	defer outFile.Close()

	fmt.Fprintf(outFile, "// Code generated by %s. DO NOT EDIT.\n", filepath.Base(os.Args[0]))
	fmt.Fprintf(outFile, "package tags\n")
	fmt.Fprintf(outFile, "var knownTags = map[string]struct{}{\n")
	for _, key := range tagKeys {
		fmt.Fprintf(outFile, "\t%q: {},\n", key)
	}
	fmt.Fprintf(outFile, "}\n")

	fmt.Fprintf(outFile, "var alternatives = map[string]string{\n")
	for _, key := range tagKeys {
		for _, alt := range append([]string{key}, tagList[key]...) {
			if alt != key {
				fmt.Fprintf(outFile, "\t%q: %q,\n", alt, key)
			}
			fmt.Fprintf(outFile, "\t%q: %q,\n", strings.ToUpper(alt), key)
			if strings.Contains(alt, "_") {
				fmt.Fprintf(outFile, "\t%q: %q,\n", strings.ReplaceAll(alt, "_", " "), key)
				fmt.Fprintf(outFile, "\t%q: %q,\n", strings.ToUpper(strings.ReplaceAll(alt, "_", " ")), key)
			}
		}
	}
	fmt.Fprintf(outFile, "}\n")
}

func cerr(err error) {
	if err != nil {
		panic(err)
	}
}
