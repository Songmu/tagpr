package gh2changelog

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	versionLinkReg    = regexp.MustCompile(`(?:^|\n)\*\*Full Changelog\*\*: (https://.*)$`)
	semverFromLinkReg = regexp.MustCompile(`.*[./](v?[0-9]+\.[0-9]+\.[0-9]+)`)
	newContribReg     = regexp.MustCompile(`(?ms)## New Contributors.*\z`)
	genCommentReg     = regexp.MustCompile(`<!-- Release notes generated using configuration.*?-->`)
)

func convertKeepAChangelogFormat(md string, d time.Time) string {
	md = strings.TrimSpace(genCommentReg.ReplaceAllString(md, ""))

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

	heading := fmt.Sprintf("## [%s](%s) - %s", semvStr, link, d.UTC().Format("2006-01-02"))
	const origHeading = "## What's Changed"
	if !strings.Contains(md, origHeading) {
		// empty
		return heading + "\n"
	}
	md = strings.Replace(md, origHeading, heading, 1)
	md = strings.ReplaceAll(md, "\n* ", "\n- ")
	md = newContribReg.ReplaceAllString(md, "")

	return strings.TrimSpace(md) + "\n"
}

var unreleasedReg = regexp.MustCompile(`(?ms)^## \[Unreleased\]\(.*?\n+(## |\z)`)

func insertNewChangelog(orig string, section string) string {
	orig = strings.TrimSpace(orig) + "\n"
	orig = unreleasedReg.ReplaceAllString(orig, `$1`)
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
