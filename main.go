package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"syscall"
)

var blastTools = map[string]string{
	"blastn":   "/usr/bin/blastn",
	"blastp":   "/usr/bin/blastp",
	"blastx":   "/usr/bin/blastx",
	"tblastn":  "/usr/bin/tblastn",
	"tblastx":  "/usr/bin/tblastx",
	"blastall": "/usr/local/bin/blastall",
}

/*
BLASTDB=tmp/blastdb/ ./myblast /usr/local/bin/blastall -a 2 -b 150 -v 150 -e 1e-10 -p blastn -F F -r 2 -W 11 -q 3 -I T -d tmp/blastdb/public-2019-05-10 -m9 -i tmp/blastdb/query.fa
*/

// setup the path for the blast tools
func init() {
	for t := range blastTools {
		if _, err := os.Stat(blastTools[t]); err != nil {
			fmt.Fprintf(os.Stderr, "init(): Tool %s not found at %s\n", t, blastTools[t])
		}
	}
}

const defaultFailedCode = 1

func runCommand(name string, args ...string) (stdout string, stderr string, exitCode int) {
	//log.Println("run command:", name, args)
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	err := cmd.Run()
	stdout = outbuf.String()
	stderr = errbuf.String()

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			//log.Printf("Could not get exit code for failed program: %v, %v", name, args)
			exitCode = defaultFailedCode
			if stderr == "" {
				stderr = err.Error()
			}
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	//log.Printf("command result, stdout: %v, stderr: %v, exitCode: %v", stdout, stderr, exitCode)
	return stdout, stderr, exitCode
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error. missing arguments")
		os.Exit(1)
	}
	tool := path.Base(os.Args[0])
	if tool == "main" {
		tool = "blastn"
	}

	if _, ok := blastTools[tool]; ok == false {
		fmt.Fprintf(os.Stderr, "Error. Unknown tool called %s\n", tool)
		os.Exit(1)
	}

	var inputFile string
	hasher := sha256.New()

	toolArgs := []string{}
	if len(os.Args) > 1 {
		var idx int
		for i, a := range os.Args[1:] {
			//fmt.Printf(" %d\t%s\n", i, a)
			toolArgs = append(toolArgs, a)
			if a == "-i" || a == "-query" {
				idx = i + 2
			} else {
				hasher.Write([]byte(a))
			}
		}

		// add the contents of the input file to our hash
		if idx < len(os.Args) {
			inputFile = os.Args[idx]
			fileData, err := ioutil.ReadFile(inputFile) // just pass the file name
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			hasher.Write(fileData)
		} else {
			fmt.Println("query file not specified")
			os.Exit(1)
		}
	}

	// find result in cache
	// build key

	fmt.Println("input: ", inputFile)
	cacheKey := fmt.Sprintf("%x", hasher.Sum(nil))
	fmt.Println("cache_key: ", cacheKey)

	return

	stOut, stErr, exitCode := runCommand(blastTools[tool], toolArgs...)
	fmt.Fprintf(os.Stderr, stErr)
	fmt.Fprintf(os.Stdout, stOut)

	// store results in our cache

	os.Exit(exitCode)
}
