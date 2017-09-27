package fns

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type AdaptiveTable struct {
	columns []atCol
	lines [][]string
	haveLabels bool
}

type atCol struct {
	leading, width int
	align int
	label string
}

func NewAdaptiveTable(spec string) *AdaptiveTable {
	var cols []atCol
	var leading int
	for _, c := range spec {
		switch c {
		case 'l':
			cols = append(cols, atCol{leading, 0, -1, ""})
			leading = 0
		case 'c':
			cols = append(cols, atCol{leading, 0, 0, ""})
			leading = 0
		case 'r':
			cols = append(cols, atCol{leading, 0, 1, ""})
			leading = 0
		default:
			leading++
		}
	}
	return &AdaptiveTable{cols, [][]string{}, false}
}

func (at *AdaptiveTable) SetLabels(labels... string) {
	for lX, label := range labels {
		if lX < len(at.columns) {
			at.columns[lX].label = label
		}
	}
	at.haveLabels = true
}

func (at *AdaptiveTable) Print(args... interface{}) {
	numCols := len(at.columns)
	if len(args) < numCols {
		numCols = len(args)
	}
	lX := 0
	for {
		filled := 0
		line := make([]string, numCols)
		for cX := 0; cX < numCols; cX++ {
			arg := args[cX]
			switch arg.(type) {
			case string:
				if lX < 1 {
					line[cX] = arg.(string)
					filled++
				}
			case []string:
				if lX < len(arg.([]string)) {
					line[cX] = arg.([]string)[lX]
					filled++
				}
			}
		}
		if filled < 1 {
			break
		}
		at.lines = append(at.lines, line)
		lX++
	}
}

func (at *AdaptiveTable) Flush() {
	for _, line := range at.lines {
		for aX, cell := range line {
			wid := utf8.RuneCountInString(cell)
			if wid > at.columns[aX].width {
				at.columns[aX].width = wid
			}
		}
	}
	padding := "                                        "
	if at.haveLabels {
		line1 := make([]string, len(at.columns))
		line2 := make([]string, len(at.columns))
		for cX, col := range at.columns {
			if col.width > 0 {
				if len(col.label) > col.width {
					at.columns[cX].width = len(col.label)
				}
				line1[cX] = col.label
				line2[cX] = strings.Repeat("=", len(col.label))
			}
		}
		newLines := [][]string{line1, line2}
		newLines = append(newLines, at.lines...)
		at.lines = newLines
	}
	for _, line := range at.lines {
		nextBefore := 0
		for cX, col := range at.columns {
			cell := line[cX]
			after := col.width - utf8.RuneCountInString(cell)
			if after < 0 {
				after = 0
			}
			before := 0
			switch col.align {
			case 0:
				before = after / 2
				after -= before
			case 1:
				before, after = after, before
			}
			before += col.leading + nextBefore
			nextBefore = after
			fmt.Print(padding[:before] + cell)
		}
		fmt.Println()
	}
}

