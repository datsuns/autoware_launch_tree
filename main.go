package main

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

var (
	launchFilePattern = regexp.MustCompile(".*\\.launch\\.xml")
)

type LaunchXmlFormat struct {
	XMLName xml.Name `xml:"launch"`
	Arg     []struct {
		Name    string `xml:"arg,attr"`
		Default string `xml:"default,attr"`
	} `xml:"arg"`
	Group []struct {
		If      string `xml:"if,attr"`
		Include []struct {
			File string `xml:"file,attr"`
		} `xml:"include"`
	} `xml:"group"`
}

type LaunchXml struct {
	FullPath     string
	Name         string
	Includes     []string
	IncludedFrom []string
}

func find(dir string) []string {
	ret := []string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !launchFilePattern.MatchString(path) {
			return nil
		}
		ret = append(ret, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return ret
}

func parseXml(path string) LaunchXml {
	ret := LaunchXml{
		FullPath:     path,
		Name:         filepath.Base(path),
		Includes:     []string{},
		IncludedFrom: []string{},
	}
	var x LaunchXmlFormat
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	xml.Unmarshal(raw, &x)
	for _, g := range x.Group {
		for _, i := range g.Include {
			//fmt.Println(filepath.Base(i.File))
			ret.Includes = append(ret.Includes, filepath.Base(i.File))
		}
	}
	return ret
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	root := filepath.Join(home, "autoware", "install")

	fmt.Println("search xml", root)
	xmls := find(root)
	if err != nil {
		panic(err)
	}

	files := []LaunchXml{}
	for _, x := range xmls {
		files = append(files, parseXml(x))
	}

	dot, err := os.Create("graph.dot")
	if err != nil {
		panic(err)
	}
	defer dot.Close()
	dot.WriteString("digraph graph_name {\n")
	dot.WriteString("    graph [\n")
	//dot.WriteString("        layout = circo\n")
	//dot.WriteString("        layout = dot\n")
	dot.WriteString("        layout = fdp\n")
	//dot.WriteString("        layout = neato\n")
	//dot.WriteString("        layout = osage\n")
	//dot.WriteString("        layout = sfdp\n")
	//dot.WriteString("        layout = twopi\n")
	dot.WriteString("    ]\n")
	for _, f := range files {
		for _, inc := range f.Includes {
			dot.WriteString(
				fmt.Sprintf("    \"%v\" -> \"%v\";\n", f.Name, inc),
			)
		}
	}
	dot.WriteString("}\n")
}
