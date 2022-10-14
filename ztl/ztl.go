package ztl

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ktr0731/go-fuzzyfinder"
)

const (
	ZETTELBOX = "/home/juergen/schreibstube/zettelkasten"
	EDITOR    = "/usr/bin/textadept"
)

type ZettelData struct {
	ZettelBox     string
	Editor        string
	EditorOptions []string
}

func InitZettelData() (*ZettelData, error) {
	zd := &ZettelData{
		ZettelBox:     ZETTELBOX,
		Editor:        EDITOR,
		EditorOptions: []string{"-n", "-f"},
	}
	// check if DirMemoriae exists - and create it if not
	fi, err := os.Stat(zd.ZettelBox)
	if err != nil {
		// dir does not exist - create it:
		err = os.MkdirAll(zd.ZettelBox, 0755)
		if err != nil {
			return nil, fmt.Errorf("could not create memoriae dir %v: %v", zd.ZettelBox, err)
		}
	} else {
		if !fi.IsDir() {
			// dir does not exist - create it:
			err = os.MkdirAll(zd.ZettelBox, 0644)
			if err != nil {
				return nil, fmt.Errorf("could not create memoriae dir %v: %v", zd.ZettelBox, err)
			}
		}
	}

	return zd, nil
}

// NewZtl creates new file; if no filepath is given, creates ztl with default ztl-name:
// ztl-YYYY-MM-DD.md
func (zd *ZettelData) NewZtl(args []string) error {
	var filename string
	// create filename
	if len(args) == 0 {
		filename = fmt.Sprintf("ztl-%s.md", time.Now().Format("2006-01-02_150405"))
	} else {
		if strings.HasSuffix(args[0], ".md") {
			filename = args[0]
		} else {
			filename = fmt.Sprintf("%s.md", args[0])
		}
	}

	// create filepath
	fp := path.Join(zd.ZettelBox, filename)

	/*
		// check if filepath already exists - and create it if not...
		_, err := os.Stat(fp)
		if err != nil {
			// file does not exist - create it:

				var txt string
				if strings.HasPrefix(filename, "ztl-") {
					txt = fmt.Sprintf("# %s\n", filename[4:len(filename)-3])
				} else {
					txt = fmt.Sprintf("# %s\n", filename[:len(filename)-3])
				}

			err = os.WriteFile(fp, []byte{}, 0644)
			if err != nil {
				return fmt.Errorf("could not create and write file %v: %v", fp, err)
			}
		}
	*/

	editorCmd := exec.Command(zd.Editor, fp, "-n", "-f", "-e", "newztl")
	return editorCmd.Start()

}

func (zd *ZettelData) OpenZtl() error {
	// get zettels
	ztls, err := zd.GetFilelist()
	if err != nil {
		return err
	}
	// search for h1-header
	args := []string{`^# `}
	rec, err := Search(args, ztls, false)
	if err != nil {
		return err
	}
	idx, err := fuzzyfinder.Find(
		rec,
		func(i int) string {
			p := strings.SplitN(rec[i], ":", 3)
			return p[2]
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			p := strings.SplitN(rec[i], ":", 3)
			s, _ := os.ReadFile(p[0])
			return string(s)
		}))
	if err != nil {
		log.Fatal(err)
	}
	p := strings.SplitN(rec[idx], ":", 3)
	return zd.OpenFile(p[0], p[1])
}

// Search searches in given files for args and return list
// uses regex-format for arg-Input
// listformat: filepath:linenumber:line
func Search(args []string, filelist []string, isAND bool) ([]string, error) {
	rec := []string{}

	for _, fp := range filelist {
		f, err := os.OpenFile(fp, os.O_RDONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("could open file to read %v: %v", fp, err)
		}
		scanner := bufio.NewScanner(f)
		idx := 1
		matchCount := 0
		for scanner.Scan() {
			for _, arg := range args {
				reg := regexp.MustCompile(arg)
				if reg.MatchString(scanner.Text()) {
					if isAND {
						matchCount++
					} else {
						rec = append(rec, fmt.Sprintf("%s:%d:%s", fp, idx, scanner.Text()))
						break
					}
				}
			}
			if isAND {
				if matchCount == len(args) {
					rec = append(rec, fmt.Sprintf("%s:%d:%s", fp, idx, scanner.Text()))
				}
			}
			idx++
		}
	}

	return rec, nil
}

func (zd *ZettelData) GetFilelist() ([]string, error) {
	dirContent, err := os.ReadDir(zd.ZettelBox)
	if err != nil {
		return nil, fmt.Errorf("could not read dir %v: %v", zd.ZettelBox, err)
	}

	filelist := []string{}

	for _, f := range dirContent {
		if !f.IsDir() {
			filelist = append(filelist, path.Join(zd.ZettelBox, f.Name()))
		}
	}

	return filelist, nil
}

func (zd *ZettelData) OpenFile(fp string, line string) error {
	editorCmd := exec.Command(zd.Editor, fp, "-n", "-f", "-l", line)
	return editorCmd.Start()
}

func (zd *ZettelData) Find(args []string, isAND bool) error {
	filelist, err := zd.GetFilelist()
	if err != nil {
		return (err)
	}

	list, err := Search(args, filelist, false)
	if err != nil {
		return (err)
	}
	idx, err := fuzzyfinder.Find(
		list,
		func(i int) string {
			return strings.SplitN(list[i], ":", 3)[2]
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			fp := strings.SplitN(list[i], ":", 3)[0]
			s, _ := os.ReadFile(fp)
			return string(s)
		}))

	if err != nil {
		return (err)
	}

	rec := strings.SplitN(list[idx], ":", 3) // 0: filepath, 1:line number, 2: line
	/*
		lineNumber, err := strconv.Atoi(rec[1])
		if err != nil {
			return fmt.Errorf("could not convert string to int %s: %v", rec[1], err)
		}
	*/
	// fmt.Printf("selected: %v\n", list[idx])
	return zd.OpenFile(rec[0], rec[1])

}

// Get ZtlHeader returns the ##-Header for the given linenumber or empty string if not found
func GetZtlHeader(fp string, linenumberString string) (string, error) {
	linenumber, err := strconv.Atoi(linenumberString)
	if err != nil {
		return "", fmt.Errorf("could not convert linenumber-string to int %s: %v", linenumberString, err)
	}
	if linenumber <= 0 {
		return "", fmt.Errorf("linenumber out of range: %v", linenumber)
	}
	linenumber-- // line in files start on 1; in "Arrays" at 0...
	bs, err := os.ReadFile(fp)
	if err != nil {
		return "", fmt.Errorf("could not open file %v: %v", fp, err)
	}

	lines := strings.Split(string(bs), "\n")
	if linenumber >= len(lines) {
		return "", fmt.Errorf("linenumber out of range %d: lines %d", linenumber, len(lines))
	}

	for i := linenumber; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "# ") {
			return lines[i], nil
		}
	}
	return "", nil
}
