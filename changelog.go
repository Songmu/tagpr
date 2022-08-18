package rcpr

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Songmu/flextime"
)

var (
	versionLinkReg    = regexp.MustCompile(`\n\*\*Full Changelog\*\*: (https://.*)$`)
	semverFromLinkReg = regexp.MustCompile(`.*[./](v?[0-9]+\.[0-9]+\.[0-9]+)`)
	newContribReg     = regexp.MustCompile(`(?ms)## New Contributors.*\z`)
)

func convertKeepAChangelogFormat(md string) string {
	md = strings.TrimSpace(md)

	var link string
	md = versionLinkReg.ReplaceAllStringFunc(md, func(match string) string {
		m := versionLinkReg.FindStringSubmatch(match)
		link = m[1]
		return ""
	})
	var semvStr string
	if m := semverFromLinkReg.FindStringSubmatch(link); len(m) > 1 {
		semvStr = m[1]
	}
	now := flextime.Now()

	heading := fmt.Sprintf("## [%s](%s) - %s", semvStr, link, now.Format("2006-01-02"))
	md = strings.Replace(md, "## What's Changed", heading, 1)
	md = strings.ReplaceAll(md, "\n* ", "\n- ")
	md = newContribReg.ReplaceAllString(md, "")

	return strings.TrimSpace(md) + "\n"
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

var changelogReg = regexp.MustCompile(`(?i)^# Change\s?log`)

func insertNewChangelog(orig string, section string) string {
	orig = strings.TrimSpace(orig) + "\n"
	section = strings.TrimSpace(section) + "\n"

	var bf bytes.Buffer
	lineSnr := bufio.NewScanner(strings.NewReader(orig))
	inserted := false
	for lineSnr.Scan() {
		line := lineSnr.Text()
		if !inserted && strings.HasPrefix(line, "## ") {
			bf.WriteString(section)
			bf.WriteString("\n")
			inserted = true
		}
		bf.WriteString(line)
		bf.WriteString("\n")
	}
	if !inserted {
		bf.WriteString("\n")
		bf.WriteString(section)
	}
	return bf.String()
}
