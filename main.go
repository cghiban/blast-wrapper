package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"
)

var storeDir = "/tmp/blastCacheStore"

var blastTools = map[string]string{
	"blastn":  "/usr/bin/blastn",
	"blastp":  "/usr/bin/blastp",
	"blastx":  "/usr/bin/blastx",
	"tblastn": "/usr/bin/tblastn",
	"tblastx": "/usr/bin/tblastx",
	//"blastall": "/usr/local/bin/blastall",
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
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return "", "", nil
	}

	// is is safe to ignore these errors?
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

func buildHashKey(args []string) (key string, err error) {
	input := ""
	idx := 0
	// 1st pass, find the input
	for i := 1; i < len(args); i++ {
		if args[i] == "-query" {
			idx = i
			break
		}
	}

	if idx > 0 && idx < (len(args)-1) && args[idx+1] != "" {
		//fmt.Printf("input file=%s\n", args[idx+1])
		input = args[idx+1]
	} else {
		//fmt.Print("Error: missing input file")
		return "", errors.New("missing input file")
	}

	if _, err := os.Stat(input); os.IsNotExist(err) {
		fmt.Println("Error: Input file not found")
		return "", errors.New("missing input file")
	} else if err != nil {
		fmt.Println("Error:", err)
		return "", errors.New("missing input file")
	}
	//fmt.Printf("found input [%s] at index [%d]\n", input, idx)

	// 2nd pass, compute hash key
	// also, don't compute the key if -h or --help flags are found
	h := md5.New()

	// also include the tool when computing the hash
	io.WriteString(h, path.Base(args[0]))
	for i := 1; i < len(args); i++ {
		if args[i] == "-h" || args[i] == "-help" || args[i] == "--help" {
			return "", errors.New("-help flag was set")
		} else if i != idx && i != (idx+1) {
			io.WriteString(h, args[i])
		}
	}
	// now add the file
	f, err := os.Open(input)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	hashKey := fmt.Sprintf("%x", h.Sum(nil))
	return hashKey, nil
}

func main() {

	tool := path.Base(os.Args[0])
	if tool == "main" {
		tool = "blastn"
	}

	if _, ok := blastTools[tool]; ok == false {
		fmt.Fprintf(os.Stderr, "Error. Unknown tool called %s\n", tool)
		os.Exit(1)
	}

	toolArgs := os.Args[1:]

	// when returned err is not nil, we pass all the args to the blast tool
	cacheKey, err := buildHashKey(os.Args)

	/*
		fmt.Fprintf(os.Stderr, "cache_key: %s\n", cacheKey)
		fmt.Fprintln(os.Stderr, toolArgs)
		return
	*/

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
