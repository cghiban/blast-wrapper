package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"syscall"
)

var storeDir = "/tmp/blastCacheStore"

var blastTools = map[string]string{
	"blastn":   "/usr/bin/blastn",
	"blastp":   "/usr/bin/blastp",
	"blastx":   "/usr/bin/blastx",
	"tblastn":  "/usr/bin/tblastn",
	"tblastx":  "/usr/bin/tblastx",
	"blastall": "/usr/local/bin/blastall",
}

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

func findInStore(key string) (output string, warnings string, err error) {

	if key == "" || len(key) < 10 {
		return "", "", errors.New("invalid key given")
	}

	cacheDir := fmt.Sprintf("%s/%s/%s/%s", storeDir, key[0:3], key[3:6], key)
	//fmt.Println("cache dir: ", cacheDir)
	// TODO
	// XXXX - check if cahce dir exists
	/*if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", "", err
	}*/

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return "", "", nil
	}

	blastOutput, err := ioutil.ReadFile(cacheDir + "/output.blast")
	blastErrors, err := ioutil.ReadFile(cacheDir + "/errors.blast")

	return string(blastOutput), string(blastErrors), nil
}

func addToStore(key string, blastOutput string, blastErrors string) (err error) {

	if key == "" || len(key) < 10 {
		return errors.New("invalid key given")
	}

	cacheDir := fmt.Sprintf("%s/%s/%s/%s", storeDir, key[0:3], key[3:6], key)
	//fmt.Println("cache dir: ", cacheDir)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	// func WriteFile(filename string, data []byte, perm os.FileMode) error

	if err := ioutil.WriteFile(cacheDir+"/output.blast", []byte(blastOutput), 0644); err != nil {
		return err
	}

	if err := ioutil.WriteFile(cacheDir+"/errors.blast", []byte(blastErrors), 0644); err != nil {
		return err
	}

	return err
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
			//fmt.Fprintf(os.Stderr, " %d\t%d\t%s\n", i, idx, a)
			toolArgs = append(toolArgs, a)
			if a == "-i" || a == "-query" {
				idx = i + 2
			} else if i+1 != idx { // skip the filename
				//fmt.Fprintf(os.Stderr, "\tadded %d\t%d\t%s\n", i, idx, a)
				hasher.Write([]byte(a))
			}
		}

		// add the contents of the input file to our hash
		if idx < len(os.Args) {
			inputFile = os.Args[idx]
			fileData, err := ioutil.ReadFile(inputFile) // just pass the file name
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}
			hasher.Write(fileData)
		} else {
			fmt.Fprint(os.Stderr, "query file not specified")
			os.Exit(1)
		}
	}

	// build key

	//fmt.Println("input: ", inputFile)
	cacheKey := fmt.Sprintf("%x", hasher.Sum(nil))
	//fmt.Fprintf(os.Stderr, "cache_key: %s\n", cacheKey)

	//return

	// find result in cache
	blastOutput, blastErrors, err := findInStore(cacheKey)
	/*fmt.Println("blastOutput: ", blastOutput)
	fmt.Println("blastErrors:", blastErrors)
	fmt.Println("err:", err)
	*/

	exitCode := 0
	if err != nil || blastOutput == "" {
		blastOutput, blastErrors, exitCode = runCommand(blastTools[tool], toolArgs...)

		// add to cache
		if exitCode == 0 {
			addToStore(cacheKey, blastOutput, blastErrors)
		}
	}

	fmt.Fprint(os.Stdout, blastOutput)
	fmt.Fprint(os.Stderr, blastErrors)

	os.Exit(exitCode)
}
