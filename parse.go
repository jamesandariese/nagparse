package nagparse

import (
	"bufio"
	"io"
	"strings"
)

type NagiosObject struct {
	name       string
	properties map[string]string
}

func isBlankLine(line string) bool {
	trimmed_string := strings.TrimSpace(line)
	if len(trimmed_string) == 0 {
		return true
	}
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return true
	}
	return false
}

func isBlockClose(line string) bool {
	return strings.TrimSpace(line) == "}"
}

func getBlock(blockName string, isStatusFile bool, lines <-chan string, out chan<- NagiosObject, typ reflect.Type) {
	obj := NagiosObject{name: blockName, properties: make(map[string]string, 40)}
	splitter := ' '
	if isStatusFile {
		splitter = '='
	}
	for {
		line, ok := <-lines
		if !ok {
			return
		}
		if isBlankLine(line) {
			continue
		}
		if isBlockClose(line) {
			out <- obj
                        return
		}
                line = strings.TrimSpace(line)
		point := strings.IndexRune(line, splitter)
                if point == -1 {
			panic("need a key and value pair to unpack: "+line)
		}
		obj.properties[line[:point]] = line[point:]
	}
}

func getBlockBeginning(line string, lines <-chan string, out chan<- NagiosObject, typ reflect.Type) {
	trimmed_string := strings.TrimSpace(line)

	if !strings.HasSuffix(trimmed_string, "{") {
		panic("did not find block beginning (line ending in {) where expected")
	}

	trimmed_string = strings.TrimSpace(line[:len(line)-1])
	if strings.IndexAny(trimmed_string, "\t ") != -1 {
		getBlock(trimmed_string, false, lines, out, typ)
	} else {
		getBlock(trimmed_string, true, lines, out, typ)
	}
}

func parseLines(lines <-chan string, out chan<- NagiosObject, typ reflect.Type) {
	for {
		line, ok := <-lines
		if !ok {
			return
		}
		if isBlankLine(line) {
			continue
		}
		getBlockBeginning(line, lines, out, typ)

	}
}

func Parse(in io.Reader, out chan<- NagiosObject, typ interface{}) error {
	defer func() {
		close(out)
	}()
	lines := make(chan string)
	go parseLines(lines, out, reflect.TypeOf(typ))
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
        close(lines)
	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}
	return nil
}
