package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

func main() {
	logPath := "../logs/jobs.log"
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	pipePath := "../pipes/jobPushPipe"
	pullPath := "../pipes/jobPullPipe"
	createFIFO(pipePath)
	createFIFO(pullPath)
	args := os.Args
	if len(args) < 2 {
		fmt.Println("-> Invalid command! Use 'jobs help' to get the list of available commands")
		os.Exit(1)
	}
	switch args[1] {
	case "help":
		if len(args) > 2 {
			fmt.Println("-> Incorrect command! 'jobs help' takes no arguments")
			os.Exit(1)
		}
		fmt.Println("Usage:")
		fmt.Println("   jobs <command> <arguments>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("   help -> lists all the available commands")
		fmt.Println("   push -> push a job or a list of jobs mentioned via a file to be completed")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("   -f -> specify a file after it to pull the list of jobs")
		os.Exit(0)

	case "push":
		if len(args) == 2 {
			fmt.Println("-> Incorrect command! 'jobs push' takes atleast 1 argument and 2 at max.")
			os.Exit(1)
		}
		startDaemon()
		var jobList []string = []string{}
		if args[2] == "-f" {
			err := readFile(args[3], &jobList)
			if err != nil {
				log.Println("-> Error reading file: ", err)
				fmt.Println("-> Error reading file: ", err)
			}
			pushToPipe(pipePath, &jobList)
		} else {
			jobList = append(jobList, strings.Join(args[2:], " "))
			pushToPipe(pipePath, &jobList)
		}
		go readFromPipe(len(jobList), pullPath, wg)
	}
	wg.Wait()
}

func readFile(filePath string, list *[]string) error {
	readFile, err := os.Open(filePath)

	if err != nil {
		return err
	}
	fileScanner := bufio.NewScanner(readFile)

	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		*list = append(*list, fileScanner.Text())
	}

	readFile.Close()
	return nil
}

func pushToPipe(pipe string, list *[]string) {
	f, err := os.OpenFile(pipe, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Println("Error opening pipe: ", err)
		fmt.Printf("error opening pipe: %v\n", err)
	}
	for _, j := range *list {
		f.WriteString(j + "\n")
	}
}

func createFIFO(path string) error {
	// Check if the FIFO already exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Create the FIFO if it doesn't exist
		err := syscall.Mkfifo(path, 0666)
		if err != nil {
			return err
		}
	}
	return nil
}

func readFromPipe(jobs int, pipe string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Logs will start to appear....")
	log.Println("Reading from pipe: " + pipe + "\n")
	fifo, err := os.OpenFile(pipe, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		log.Println("Error reading from pipe: ", err)
		fmt.Println(err)
	}
	reader := bufio.NewReader(fifo)

	for {
		line, err := reader.ReadBytes('\n')
		if err == nil {
			fmt.Print(string(line))
			log.Println("Data from pipe-" + pipe + ": " + string(line))

			if string(line)[:4] == "Done" {
				jobs -= 1
			}

			if jobs == 0 {
				log.Println("-------------------------------------------------------------------------------" + "\n")
				break
			}
		}
	}
}

func startDaemon() {
	cmd1 := exec.Command("ps", "aux")
	cmd2 := exec.Command("grep", "-e", "-d=false")
	processes := 0
	// Connect the output of cmd1 to the input of cmd2
	cmd2.Stdin, _ = cmd1.StdoutPipe()

	// Create a pipe to capture the output of cmd2
	outputPipe, _ := cmd2.StdoutPipe()

	// Start cmd2
	_ = cmd2.Start()

	// Start cmd1
	_ = cmd1.Run()

	// Read and print the output of cmd2 line by line
	scanner := bufio.NewScanner(outputPipe)
	for scanner.Scan() {
		processes += 1
	}

	// Check for any errors
	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
	}

	// Wait for cmd2 to finish
	_ = cmd2.Wait()
	if processes == 1 {
		_, err := exec.Command("go", "run", "../jobsd/main.go", "-d=true").Output()
		if err != nil {
			fmt.Println("Could not start daemon: ", err)
			log.Println("Could not start daemon: ", err)
			os.Exit(1)
		}
	}
}
