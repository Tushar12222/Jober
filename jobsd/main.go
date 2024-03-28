package main

import (
	"bufio"
	"fmt"
	_ "github.com/qodrorid/godaemon"
	"log"
	"os"
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
	log.Println("Daemon started")
	log.Println("Listening to pipe....")
	readPipe, err := os.OpenFile("../pipes/jobPushPipe", os.O_RDONLY, os.ModeNamedPipe)
	writePipe, err := os.OpenFile("../pipes/jobPullPipe", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		log.Println(err)
	}
	reader := bufio.NewReader(readPipe)

	for {
		line, err := reader.ReadBytes('\n')
		if err == nil {
			log.Print("job received: " + string(line))
			go processJob(writePipe, string(line)[:len(line)-1])
		}
	}
}

func processJob(pipe *os.File, job string) {
	pipe.WriteString("Goroutine started for the job -> " + job + "\n")
	log.Println("Goroutine started for the job -> " + job)
	log.Println("Job completed for task " + job)
	pipe.WriteString("Done with job -> " + job + "\n")
}
