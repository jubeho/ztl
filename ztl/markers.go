package ztl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/sirupsen/logrus"
)

func (zd *ZettelData) GetMarkerLists(regex string) ([]string, map[string][]string, error) {
	markerlist := []string{}
	markermap := make(map[string][]string) // key: tagname, val:"filepath:linenumber:line"
	//tagregex := `(#\w+)`

	filelist, err := zd.GetFilelist()
	if err != nil {
		return nil, nil, err
	}

	reg := regexp.MustCompile(regex)
	for _, fp := range filelist {
		f, err := os.OpenFile(fp, os.O_RDONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		scanner := bufio.NewScanner(f)
		linecount := 1
		for scanner.Scan() {
			matches := reg.FindAllString(scanner.Text(), -1)
			for _, match := range matches {
				markermap[match] = append(markermap[match], fmt.Sprintf("%s:%d:%s", fp, linecount, scanner.Text()))
			}
			//matches := reg.FindAllStringSubmatch(scanner.Text(), -1)
			//fmt.Println(matches)
			linecount++
		}
	}

	for k := range markermap {
		markerlist = append(markerlist, k)
	}

	sort.Strings(markerlist)

	return markerlist, markermap, nil
}

func (zd *ZettelData) HandleMarkers(markerlist []string, markermap map[string][]string) error {
	idx, err := fuzzyfinder.Find(
		markerlist,
		func(i int) string {
			return markerlist[i]
		},

		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			filelist := ""
			for _, l := range markermap[markerlist[i]] {
				p := strings.SplitN(l, ":", 3) // 0=filepath, 1=linenumber, 2=linevalue
				txt, err := GetZtlHeader(p[0], p[1])
				if err != nil {
					logrus.Fatal(err)
				}
				filelist = fmt.Sprintf("%s\n%s\n  %s\n    %s\n-------------", filelist, filepath.Base(p[0]), txt, p[2])
			}
			return filelist
		}))

	if err != nil {
		return (err)
	}

	list := markermap[markerlist[idx]]
	idx2, err := fuzzyfinder.Find(
		list,
		func(i int) string {
			p := strings.SplitN(list[i], ":", 3) // 0=filepath, 1=linenumber, 2=linevalue
			txt, err := GetZtlHeader(p[0], p[1])
			if err != nil {
				return ""
			}
			return fmt.Sprintf("%s [%s]", txt, p[2])
		},
	)
	if err != nil {
		return (err)
	}

	p := strings.SplitN(list[idx2], ":", 3) // 0=filepath, 1=linenumber, 2=linevalue
	return zd.OpenFile(p[0], "1")
}
