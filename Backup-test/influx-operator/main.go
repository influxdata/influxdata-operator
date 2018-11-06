package main

import (
	// "bytes"
	"flag"
	"fmt"
	// "io/ioutil"
	"log"
	// "net/http"
	"os"
	"os/exec"
	"strings"
	// "github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/session"
	// "github.com/aws/aws-sdk-go/service/s3"
)

// Personal test consts
const (
	DBName    string = "nodes"
	DBHost    string = ""
	OutputDir string = "/var/lib/influxdb/backup"
	S3_REGION        = ""
	S3_BUCKET        = ""
)

func main() {
	backup()
}

func restore(influxdatav1alpha1.Backup) {
	// For testing locally
	// restoreCmd := fmt.Sprintf("influxd restore -portable -db %s %s", DBName, OutputDir)
	restoreCmd := fmt.Sprintf("influxd restore -portable -db %s -host %s %s", DBName, DBHost, OutputDir)
	cmd := exec.Command("bash", "-c", restoreCmd)
	// Combine stdout and stderr
	printCommand(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		printError(err)
		printOutput(output)
		log.Fatal("Restore failed!")
		return
	}
	printOutput(output)
}

func backup(cr *influxdatav1alpha1.Backup) {
	// For testing locally
	// backupCmd := fmt.Sprintf("influxd backup -portable -database %s %s", DBName, OutputDir)
	backupCmd := fmt.Sprintf("influxd backup -portable -database %s %s", DBName, OutputDir)
	cmd := exec.Command("bash", "-c", backupCmd)

	// Create an *exec.Cmd
	// Combine stdout and stderr
	printCommand(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		printError(err)
		printOutput(output)
		log.Fatal("Backup failed!")
		return
	}
	printOutput(output)

	files, err := ioutil.ReadDir(OutputDir)
	if err != nil {
		log.Fatal(err)
	}

	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String(S3_REGION)})
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		file := OutputDir + "/" + f.Name()
		err := AddFileToS3(s, file)
		if err != nil {
			// TODO: Probably want to handle this in a more elegant way
			log.Fatal(err)
		}
	}
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, fileDir string) error {

	// Open the file for use
	file, err := os.Open(fileDir)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(S3_BUCKET),
		Key:                  aws.String(fileDir),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

func printCommand(cmd *exec.Cmd) {
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
}

func printError(err error) {
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("==> Error: %s\n", err.Error()))
	}
}

func printOutput(outs []byte) {
	if len(outs) > 0 {
		fmt.Printf("==> Output: %s\n", string(outs))
	}
}
