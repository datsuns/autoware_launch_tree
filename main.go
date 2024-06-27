package main

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	launchFilePattern            = regexp.MustCompile(".*\\.launch\\.xml")
	includeDynamicPackagePattern = regexp.MustCompile(".*\\$\\(var ")
	includeByPackageNamePattern  = regexp.MustCompile(".*\\$\\(find-pkg-share ")
)

type PackageXmlFormat struct {
	XMLName xml.Name `xml:"package"`
	Name    string   `xml:"name"`
}

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

type IncludeEntry struct {
	Package string
	File    string
}

type XmlFile struct {
	Name     string
	FullPath string
	Includes []IncludeEntry
}

type Package struct {
	Name     string
	FullPath string
	Location string
}

type LaunchXml struct {
	Package Package
	Xmls    []XmlFile
	Used    bool
}

func parsePackageXmls(path string) string {
	var x PackageXmlFormat
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	xml.Unmarshal(raw, &x)
	return x.Name
}

func collectPackages(root string) []Package {
	ret := []Package{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Base(path) == "package.xml" {
			n := parsePackageXmls(path)
			ret = append(ret,
				Package{Name: n, FullPath: path, Location: filepath.Dir(path)},
			)
			//fmt.Println("pkg:", path, "name", n)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return ret
}

func parseRawInclude(includeString string) string {
	root := strings.Split(includeString, "/")[0]

	if includeByPackageNamePattern.MatchString(root) {
		arg := strings.Split(root, "find-pkg-share ")[1]
		if includeDynamicPackagePattern.MatchString(arg) {
			// $(find-pkg-share $(var yyyy))
			return arg
		}

		// $(find-pkg-share xxxx)
		pkg := arg[:len(arg)-1]
		return pkg
	} else {
		// $(var xxxx)_description
		return ""
	}
}

func parseXml(path string) XmlFile {
	retXml := XmlFile{
		Name:     filepath.Base(path),
		FullPath: path,
		Includes: []IncludeEntry{},
	}
	var x LaunchXmlFormat
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	xml.Unmarshal(raw, &x)
	for _, g := range x.Group {
		for _, i := range g.Include {
			p := parseRawInclude(i.File)
			f := filepath.Base(i.File)
			retXml.Includes = append(retXml.Includes, IncludeEntry{Package: p, File: f})
		}
	}
	return retXml
}

func collectLaunchXmls(root string) []XmlFile {
	ret := []XmlFile{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !launchFilePattern.MatchString(path) {
			return nil
		}
		ret = append(ret, parseXml(path))
		return nil
	})
	if err != nil {
		panic(err)
	}
	return ret
}

func hasInclude(xmls []XmlFile) bool {
	for _, f := range xmls {
		if len(f.Includes) > 0 {
			return true
		}
	}
	return false
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	root := filepath.Join(home, "autoware", "install")

	fmt.Println("search xml", root)
	pkgs := collectPackages(root)

	xmls := []LaunchXml{}
	for _, p := range pkgs {
		xml := collectLaunchXmls(p.Location)
		xmls = append(xmls, LaunchXml{Package: p, Xmls: xml, Used: false})
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

	for _, x := range xmls {
		for _, f := range x.Xmls {
			for _, inc := range f.Includes {
				dot.WriteString(
					fmt.Sprintf("    \"%v::%v\" -> \"%v::%v\";\n", x.Package.Name, f.Name, inc.Package, inc.File),
				)
			}
		}
	}

	for i, x := range xmls {
		if x.Used {
			continue
		}
		x.Used = true
		if len(x.Xmls) == 0 {
			continue
		}
		dot.WriteString(
			fmt.Sprintf("    subgraph \"cluster_%v\"{\n", x.Package.Name),
		)
		dot.WriteString(
			fmt.Sprintf("        label=\"%v\";\n", x.Package.Name),
		)
		for _, f := range x.Xmls {
			dot.WriteString(
				fmt.Sprintf("        \"%v::%v\";\n", x.Package.Name, f.Name),
			)
		}
		for _, x2 := range xmls[i:] {
			if x.Package.Name == x2.Package.Name {
				x2.Used = true
				for _, f := range x2.Xmls {
					dot.WriteString(
						fmt.Sprintf("        \"%v::%v\";\n", x2.Package.Name, f.Name),
					)
				}
			}
		}
		dot.WriteString("    }\n")
	}

	dot.WriteString("}\n")
	fmt.Println(">>> done")
}
