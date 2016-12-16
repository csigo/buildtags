package main

import (
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	srcDir  string
	pkgDir  = flag.String("pkg_dir", ".", "package working dir")
	basePkg = flag.String("pkgs", "htc.com/csi", "package to build tags")
)

type importFile struct {
	file    string
	srcFile string
}

func init() {
	if build.Default.GOPATH == "" {
		panic("$GOPATH is empty")
	}
	for _, dir := range build.Default.SrcDirs() {
		if _, err := os.Stat(filepath.Join(dir, *basePkg)); err == nil {
			srcDir = dir
			break
		}
	}
	if srcDir == "" {
		panic(fmt.Sprintf("unable to find source dir for %s", *basePkg))
	}
}

// get imports belonged to htc.com for a file.
func getImports(filepath string) ([]string, error) {
	// Parse the file containing this very example
	// but stop after processing the imports.
	f, err := parser.ParseFile(token.NewFileSet(), filepath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	// Print the imports from the file's AST.
	result := []string{}
	for _, s := range f.Imports {
		v := s.Path.Value
		if strings.HasPrefix(v, `"`+*basePkg) {
			result = append(result, v[1:len(v)-1])
		}
	}
	return result, nil
}

func pkgToPath(pkg string) string {
	return filepath.Join(srcDir, pkg)
}

// TODO: perform real merge, but how ?!
func mergeTags(tags []string) string {
	return strings.Join(tags, " ")
}

func main() {
	flag.Parse()
	wdir, err := filepath.Abs(*pkgDir)
	if err != nil {
		panic(fmt.Sprintf("unable to get package dir of %v err:%v", wdir, err))
	}
	pkgStae, err := os.Stat(wdir)
	if err != nil {
		panic(fmt.Sprintf("unable to get dir info of %v err:%v", wdir, err))
	}
	if !pkgStae.IsDir() {
		panic(fmt.Sprintf("%v is not a dir", wdir))
	}
	tags := []string{}
	seentags := map[string]bool{}
	touched := map[string]bool{}
	queue := []importFile{importFile{file: wdir}}
	for len(queue) > 0 {
		dirname := queue[0]
		queue = queue[1:]
		// handle diamond shape cases.
		if touched[dirname.file] {
			continue
		}
		touched[dirname.file] = true
		infos, err := ioutil.ReadDir(dirname.file)
		if err != nil {
			panic(fmt.Sprintf("unable to read package %v imported from %v, err:%v",
				dirname.file, dirname.srcFile, err))
		}
		for _, info := range infos {
			if info.IsDir() {
				continue
			}
			// it go file.
			name := info.Name()
			path := filepath.Join(dirname.file, name)
			if strings.HasSuffix(name, ".go") {
				// get imports of a go file.
				imports, err := getImports(path)
				if err != nil {
					panic(fmt.Sprintf("unable to parse imports from %v err:%v", path, err))
				}
				for _, imp := range imports {
					queue = append(queue, importFile{file: pkgToPath(imp), srcFile: path})
				}
			} else if name == "go_tags" && !info.IsDir() {
				// read go tags content.
				tag, err := ioutil.ReadFile(path)
				if err != nil {
					panic(fmt.Sprintf("unable to read go_tags from %v err:%v", path, err))
				}
				stag := strings.Replace(string(tag), "\n", "", -1)
				_, seen := seentags[stag]
				if !seen {
					seentags[stag] = true
					tags = append(tags, stag)
				}
			}
		}
	}
	fmt.Print(mergeTags(tags))
}
