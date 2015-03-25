package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func resolve(imports map[string][]string, pkg string) []string {
	remaining := []string{pkg}
	seen := map[string]bool{}

	for i := 0; i < len(remaining); i++ {
		cur := remaining[i]
		if i > 0 {
			seen[cur] = true
		}

		for _, p := range imports[cur] {
			if !seen[p] {
				remaining = append(remaining, p)
			}
		}
	}

	result := []string{}
	for p := range seen {
		result = append(result, p)
	}

	return result
}

func main() {
	skipList := ""
	flag.StringVar(&skipList, "skip", "", "List of files to skip")
	flag.Parse()

	goPath := os.Getenv("GOPATH")
	base := filepath.Join(goPath, "src")

	skipMap := map[string]bool{}
	if len(skipList) > 0 {
		fns := strings.Split(skipList, ",")
		for _, fn := range fns {
			skipMap[fn] = true
		}
	}
	// Parse import deps and contained files for all the packages
	filePackage := map[string]string{}
	pkgImports := map[string][]string{}
	filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		pkgs, err := parser.ParseDir(token.NewFileSet(), path, func(fi os.FileInfo) bool {
			if skipMap[fi.Name()] {
				return false
			}
			return true
		}, parser.ImportsOnly)
		if err != nil {
			log.Panicln("couldn't parse files in src:", err.Error())
		}

		for _, pkg := range pkgs {
			importPath, err := filepath.Rel(base, path)
			if err != nil {
				log.Panicln(err.Error())
			}

			imports := []string{}
			for name, f := range pkg.Files {
				relPath, err := filepath.Rel(goPath, name)
				if err != nil {
					log.Panicln(err.Error())
				}

				filePackage[relPath] = importPath
				for _, i := range f.Imports {
					imports = append(imports, i.Path.Value[1:len(i.Path.Value)-1])
				}
			}

			pkgImports[importPath] = imports
		}

		return nil
	})

	if len(filePackage) == 0 {
		log.Panicln("No files found. Is your GOPATH set correctly?")
	}

	// Construct a map of package => all upstream imports. This could be faster if we topo-sorted the imports.
	upstreams := map[string][]string{}
	for pkg := range pkgImports {
		upstreams[pkg] = resolve(pkgImports, pkg)
	}

	// Reverse the map to pkg => all downstreams.
	downstreams := map[string][]string{}
	for pkg, up := range upstreams {
		for _, p := range up {
			downstreams[p] = append(downstreams[p], pkg)
		}
	}

	affected := map[string]bool{}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		file := scanner.Text()
		pkg := filePackage[file]

		for _, down := range downstreams[pkg] {
			affected[down] = true
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	for p := range affected {
		fmt.Println(p)
	}
}
