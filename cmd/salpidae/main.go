package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	salpidae "go-salpidae/pkg"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func getFileSize(fileName string) (int64, error) {
	f, e := os.Open(fileName)
	if e != nil {
		return 0, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return 0, e
	}
	return info.Size(), nil
}

const nrThreads int = 30

func handleFile(fileInput string, fileOutput string, blockSizeM int) error {
	blockSize := blockSizeM * 1024 * 1024
	fileSize, e := getFileSize(fileInput)
	if e != nil {
		log.Printf("Unable to read input file size: %v\n", e.Error())
		return e
	}
	file, e := os.Open(fileInput)
	if e != nil {
		log.Printf("Unable to read input file: %v\n", e.Error())
		return e
	}
	defer file.Close()
	nrBlocksPerThread := (salpidae.GetNrOfBlocks(fileSize, blockSize) / nrThreads) + 1
	signature, e := salpidae.ReadFile(file, fileSize, blockSize, nrBlocksPerThread)
	if e != nil {
		log.Printf("Unable to hash input file: %v\n", e.Error())
		return e
	}

	e = salpidae.WriteFile(fileOutput, signature)
	if e != nil {
		log.Printf("Unable to write output: %v\n", e.Error())
		return e
	}
	return nil
}

type response struct {
	Error     string
	Signature []string
}

func write(writer http.ResponseWriter, signature []string) {
	res := response{Error: "", Signature: signature}
	if jres, e := json.Marshal(res); e == nil {
		fmt.Fprintln(writer, string(jres))
	} else {
		http.Error(writer, "", http.StatusInternalServerError)
	}
}

func writeError(writer http.ResponseWriter, message string) {
	res := response{Error: message, Signature: []string{}}
	if jres, e := json.Marshal(res); e == nil {
		fmt.Fprintln(writer, string(jres))
	} else {
		http.Error(writer, "", http.StatusInternalServerError)
	}
}

func post(writer http.ResponseWriter, req *http.Request) {
	blockSizeMStr := req.PostFormValue("blocksize")
	blockSizeM, e := strconv.Atoi(blockSizeMStr)
	if e != nil {
		log.Printf("Unexpected format of block size: %s\n", e.Error())
		writeError(writer, "Unexpected format of block size")
		return
	}

	if blockSizeM <= 0 || blockSizeM > 2047 {
		log.Printf("Unsupported block size")
		writeError(writer, "Unsupported block size")
		return
	}

	file, header, e := req.FormFile("data")
	if e != nil {
		log.Printf("Unable to read data: %s", e.Error())
		writeError(writer, "Unable to read data")
		return
	}
	defer req.MultipartForm.RemoveAll()

	blockSize := blockSizeM * 1024 * 1024
	fileSize := header.Size

	nrBlocksPerThread := (salpidae.GetNrOfBlocks(header.Size, blockSize) / nrThreads) + 1
	signature, e := salpidae.ReadFile(file, fileSize, blockSize, nrBlocksPerThread)
	if e != nil {
		log.Printf("Unable to hash input file: %s", e.Error())
		writeError(writer, "Unable to hash input file")
		return
	}
	write(writer, signature)
}

func startServer(port uint) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /signature", post)
	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Aborting server: %v\n", err)
		}
	}()
	return server
}

func getLogWriter(logOutput string, def io.Writer) io.Writer {
	logWriter := def
	if len(logOutput) != 0 {
		if f, err := os.OpenFile(logOutput, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err == nil {
			logWriter = f
		}
	}
	return logWriter
}

func getLogPrefix(isFile bool, isServer bool) string {
	if isFile {
		return "[File] "
	}
	if isServer {
		return "[Server] "
	}
	return ""
}

func main() {
	var fileInput string
	flag.StringVar(&fileInput, "i", "", "file for signature generation")
	var fileOutput string
	flag.StringVar(&fileOutput, "o", "", "file for signature output")
	port := flag.Uint("s", 0, "start server on the port")
	blockSizeM := flag.Int("b", 1, "size of block in MB")
	var logOutput string
	flag.StringVar(&logOutput, "l", "", "file for log")
	flag.Parse()

	isServer := *port != 0
	isFile := len(fileInput) != 0 || len(fileOutput) != 0
	if isServer && isFile {
		fmt.Fprintf(os.Stderr, "'-s' and '-i/-o' are mutually exclusive\n")
		os.Exit(1)
	}

	if isFile {
		if len(fileInput) == 0 {
			fmt.Fprintf(os.Stderr, "'-i' input file argument is missing\n")
			os.Exit(1)
		}
		if len(fileOutput) == 0 {
			fmt.Fprintf(os.Stderr, "'-o' output file argument is missing\n")
			os.Exit(1)
		}

		if *blockSizeM <= 0 || *blockSizeM > 2047 {
			fmt.Fprintf(os.Stderr, "Unsupported block size\n")
			os.Exit(1)
		}
	}

	log.SetOutput(getLogWriter(logOutput, os.Stdout))
	log.SetPrefix(getLogPrefix(isFile, isServer))
	log.Println("**********Salpidae starting**********")

	errExit := false
	if isServer {
		server := startServer(*port)
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		log.Println("Waiting for interrupt...")
		<-stop

		log.Println("Stopping...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v\n", err.Error())
			errExit = true
		}
	} else if isFile {
		if e := handleFile(fileInput, fileOutput, *blockSizeM); e != nil {
			log.Printf("Unable to hash input file %v\n", e.Error())
			errExit = true
		}
	}
	log.Println("**********Salpidae done**********")
	if errExit {
		os.Exit(1)
	}
}
