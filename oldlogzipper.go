package main

import (
	"bufio"
	"compress/gzip"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

var (
	//go:embed oldlogzipper.cfg
	oldlogzippercfg string
	regxcontents    string

	//go:embed version.txt
	version         string
	dirname         string
	cfgfile         string
	dryrun          bool
	debug           bool
	preserveAttrs   bool
	keep            int
	dohelp          bool
	showregxdefault bool
)

func main() {
	flag.StringVar(&cfgfile, "f", "", "regex file")
	flag.BoolVar(&dryrun, "n", false, "nocompress (dry run)")
	flag.BoolVar(&debug, "D", false, "Debug log")
	flag.BoolVar(&preserveAttrs, "P", false, "going to chown/chmod as src file")
	flag.IntVar(&keep, "k", 2, "keep n files")
	flag.BoolVar(&dohelp, "h", false, "help")
	flag.BoolVar(&showregxdefault, "show_regx_default_content", false, "show regxdefault")
	flag.Parse()

	args := flag.Args()

	if showregxdefault {
		fmt.Printf("%s\n", oldlogzippercfg)
		os.Exit(0)
	}

	flag.Usage = func() {
		fmt.Printf("version %s\n", version)
		fmt.Printf("Usage: %s [options] <dir1> <dir2> ...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	if dohelp {
		flag.Usage()
	}
	if len(args) == 0 {
		flag.Usage()
	}
	for _, arg := range args {
		compressDir(arg)
	}
}

func compressDir(dirname string) {
	if !isDirectory(dirname) {
		log.Fatalf("Not a directory: %s", dirname)
	}

	if len(cfgfile) > 0 {
		regexFile, err := os.ReadFile(cfgfile)
		if err != nil {
			log.Fatal(err)
		}
		regxcontents = string(regexFile)
	} else {
		regxcontents = oldlogzippercfg
	}
	//log.Printf("ReadCfg----\n%s\n-----\n", regxcontents)

	patterns, err := readPatternsFromFile(regxcontents)
	if err != nil {
		log.Fatal(err)
	}

	files, err := getMatchingFiles(patterns, dirname)
	if err != nil {
		log.Fatal(err)
	}
	if dryrun {
		if debug {
			for _, file := range files {
				log.Printf("To compress: %s\n", file)
			}
		}
		log.Printf("%d files to compress\n", len(files))
	} else {
		for _, file := range files {
			err = compressFile(file, preserveAttrs)
			if err != nil {
				log.Printf("Failed to compress file: %s, error: %v\n", file, err)
			}
			if debug {
				log.Printf("Compress: %s\n", file)
			}
		}
		log.Printf("%d files compressed\n", len(files))
	}
}

func isDirectory(path string) bool {
	cleanPath := filepath.Clean(path)

	info, err := os.Stat(cleanPath)
	if err != nil {
		log.Fatal(err)
	}
	return info.IsDir()
}

func readPatternsFromFile(content string) ([]*regexp.Regexp, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	patterns := make([]*regexp.Regexp, 0)

	for scanner.Scan() {
		if len(scanner.Text()) == 0 {
			continue
		}
		if scanner.Text()[0] == '#' {
			continue
		}
		pattern, err := regexp.Compile(scanner.Text())
		if err != nil {
			log.Printf("regexp compil failed: %v (%s)", err, scanner.Text())
			continue
		}
		if debug {
			log.Printf("scan: %v", scanner.Text())
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

func isin(s string, a []string) bool {
	for _, v := range a {
		if s == v {
			return true
		}
	}
	return false
}

func getMatchingFiles(patterns []*regexp.Regexp, basedir string) ([]string, error) {
	files := make([][]fs.FileInfo, 0, len(patterns))
	for _, _ = range patterns {
		files = append(files, make([]fs.FileInfo, 0, 100))
	}
	linkedFileList := linkedFiles(basedir)
	opendFileList := opendFiles(basedir)
	if debug {
		for _, f := range linkedFileList {
			log.Printf("linked: %s\n", f)

		}
		for _, f := range opendFileList {
			log.Printf("opened: %s\n", f)
		}
	}

	dirEntries, err := os.ReadDir(basedir)
	if err != nil {
		log.Fatalf("Error reading the directory: %v", err)
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		matched := false
		for i, pattern := range patterns {
			if pattern.MatchString(entry.Name()) {
				fileInfo, err := entry.Info()
				if err != nil {
					log.Printf("Error getting file info for %v: %v", entry, err)
					continue
				}

				files[i] = append(files[i], fileInfo)
				// log.Printf("matched: %s\n", path)
				matched = true
				break
			}
		}
		if !matched {
			log.Printf("NOmatch: %s\n", entry.Name())
		}
	}
	if err != nil {
		return nil, err
	}
	for i, l := range files {
		newl := make([]fs.FileInfo, 0, len(l))
		if len(l) <= keep {
			for _, fi := range l {
				log.Printf("keep (new): %s\n", fi.Name())
			}
			files[i] = newl
			continue
		}
		sort.Slice(l, func(i, j int) bool {
			return l[i].ModTime().Before(l[j].ModTime())
		})
		for _, fi := range l[len(l)-keep:] {
			log.Printf("keep (new): %s\n", fi.Name())
		}
		sortedl := l[:len(l)-keep]
		for _, fi := range sortedl {
			path := filepath.Join(basedir, fi.Name())
			if isin(path, linkedFileList) {
				log.Printf("keep (linked): %s\n", fi.Name())
				continue
			}
			if isin(path, opendFileList) {
				log.Printf("keep (opened): %s\n", fi.Name())
				continue
			}
			newl = append(newl, fi)
		}
		files[i] = newl
	}
	rslt := make([]string, 0, 100)
	for _, l := range files {
		for _, fi := range l {
			path := filepath.Join(basedir, fi.Name())
			rslt = append(rslt, path)
		}
	}
	return rslt, nil
}

func linkedFiles(dir string) []string {
	linkedfiles := make([]string, 0, 10)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return linkedfiles
	}

	for _, file := range files {
		if file.Mode()&os.ModeSymlink != 0 {
			linkPath, err := os.Readlink(filepath.Join(dir, file.Name()))
			if err != nil {
				continue
			}
			absPath, err := filepath.Abs(linkPath)
			if err != nil {
				continue
			}
			linkedfiles = append(linkedfiles, absPath)
		}
	}

	return linkedfiles
}
func compressFile(src string, preserveAttrs bool) error {
	baseName := filepath.Base(src)
	dirName := filepath.Dir(src)

	dst := baseName + ".gz"
	i := 1

	for {
		_, err := os.Stat(filepath.Join(dirName, dst))
		if os.IsNotExist(err) {
			break
		}
		dst = fmt.Sprintf("%s.duplicated%d.gz", baseName, i)
		i++
	}
	tmpfile := dst + ".tmp"
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	tmpFile, err := os.Create(filepath.Join(dirName, tmpfile))
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	gw := gzip.NewWriter(tmpFile)

	_, err = io.Copy(gw, srcFile)
	if err != nil {
		srcFile.Close()
		gw.Close()
		os.Remove(tmpFile.Name())
		return err
	}
	srcFile.Close()
	gw.Close()
	err = os.Rename(tmpFile.Name(), filepath.Join(dirName, dst))
	if err != nil {
		os.Remove(tmpFile.Name())
		return err
	}

	if preserveAttrs {
		srcFile, err := os.Open(src)
		if err != nil {
			return err
		}
		srcFileInfo, err := srcFile.Stat()
		if err != nil {
			srcFile.Close()
			return err
		}
		srcFile.Close()
		err = os.Remove(src)
		if err != nil {
			return err
		}
		dstFile, err := os.OpenFile(filepath.Join(dirName, dst), os.O_WRONLY, srcFileInfo.Mode())
		if err != nil {
			return err
		}

		err = os.Chmod(dstFile.Name(), srcFileInfo.Mode())
		if err != nil {
			return err
		}

		err = os.Chown(dstFile.Name(), int(srcFileInfo.Sys().(*syscall.Stat_t).Uid), int(srcFileInfo.Sys().(*syscall.Stat_t).Gid))
		if err != nil {
			return err
		}

	} else {
		err = os.Remove(src)
		if err != nil {
			return err
		}
	}

	return nil
}

func opendFiles(dir string) []string {
	fileList := make([]string, 0, 10)
	procs, err := ioutil.ReadDir("/proc")
	if err != nil {
		return fileList
	}
	openedFiles := make(map[string]bool)

	for _, proc := range procs {
		if !proc.IsDir() {
			continue
		}

		pid := proc.Name()
		fdsDir := filepath.Join("/proc", pid, "fd")

		fds, err := ioutil.ReadDir(fdsDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			fdPath := filepath.Join(fdsDir, fd.Name())
			link, err := os.Readlink(fdPath)
			if err != nil {
				continue
			}

			absLink, err := filepath.Abs(link)
			if err != nil {
				continue
			}

			if strings.HasPrefix(absLink, dir) {
				openedFiles[absLink] = true
			}
		}
	}

	for file := range openedFiles {
		fileList = append(fileList, file)
	}

	return fileList
}

func getInodeFromLink(linkPath string) (uint64, error) {
	linkDest, err := os.Readlink(linkPath)
	if err != nil {
		return 0, err
	}

	linkDestAbs := filepath.Join(filepath.Dir(linkPath), linkDest)
	var stat syscall.Stat_t
	if err := syscall.Stat(linkDestAbs, &stat); err != nil {
		return 0, err
	}

	return stat.Ino, nil
}
