package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type RequestBody struct {
	Username string `json:"Username"`
}

type Response struct {
	Username           string `json:"Username"`
	Display            string `json:"Display"`
	IfExistsResult     int    `json:"IfExistsResult"`
	IsUnmanaged        bool   `json:"IsUnmanaged"`
	ThrottleStatus     int    `json:"ThrottleStatus"`
	IsSignupDisallowed bool   `json:"IsSignupDisallowed"`
}

type Result struct {
	Email              string `json:"email"`
	Valid              bool   `json:"valid"`
	Display            string `json:"display,omitempty"`
	IsUnmanaged        bool   `json:"is_unmanaged,omitempty"`
	ThrottleStatus     int    `json:"throttle_status,omitempty"`
	IsSignupDisallowed bool   `json:"is_signup_disallowed,omitempty"`
}

func main() {
	var file *string
	var output *string
	var exists *bool

	file = flag.String("i", "", "Specify input file with email addresses.")
	output = flag.String("o", "", "Specify the output file to write results into.")
	exists = flag.Bool("e", false, "Only write valid email addresses to the output file.")

	flag.Parse()

	var data []string
	results := ChkStdin(os.Stdin)

	if len(results) > 0 {
		data = append(data, strings.TrimSpace(results))
	} else if *file != "" {
		data = ReadFile(file)
	} else {
		usage()
		os.Exit(1)
	}

	var wg sync.WaitGroup
	emailChannel := make(chan string)
	numWorkers := 5
	rateLimit := time.NewTicker(3 * time.Second) // Rate limiter

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go func() {
			for email := range emailChannel {
				response, err := validateEmail(email)
				if err != nil {
					log.Printf("[!]Error validating email %s: %v", email, err)
					wg.Done()
					continue
				}

				if *output != "" {
					if *exists {
						if response.Valid {
							FileWrite([]byte(response.Email), *output)
						}
					} else {
						FileWrite(CreateJson(response), *output)
					}
				} else {
					if *exists {
						if response.Valid {
							fmt.Println(response.Email)
						}
					} else {
						fmt.Println(string(CreateJson(response)))
					}
				}

				wg.Done() // Notify that this goroutine is done
			}
		}()
	}

	// Send emails to the channel with rate limiting
	for _, email := range data {
		wg.Add(1) // Increment the WaitGroup counter
		emailChannel <- email
		<-rateLimit.C // Wait for the rate limiter before sending the next email
	}
	close(emailChannel) // Close the channel after sending all emails

	wg.Wait() // Wait for all goroutines to finish
	fmt.Println("[*] All emails processed.")
}

func validateEmail(email string) (Result, error) {
	url := "https://login.microsoftonline.com/common/GetCredentialType"
	body := RequestBody{Username: email}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}

	// Create a new HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a timeout for the request
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("[!] Failed to validate email: %s", resp.Status)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Result{}, err
	}

	valid := response.IfExistsResult == 0 // 0 means valid
	result := Result{
		Email:              email,
		Valid:              valid,
		Display:            response.Display,
		IsUnmanaged:        response.IsUnmanaged,
		ThrottleStatus:     response.ThrottleStatus,
		IsSignupDisallowed: response.IsSignupDisallowed,
	}

	return result, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s: ", os.Args[0])
	fmt.Println()
	flag.PrintDefaults()
}

func ChkStdin(stdin *os.File) string {
	var results string

	stat, err := stdin.Stat()
	if err != nil {
		log.Fatal(err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		bytes, err := io.ReadAll(stdin)
		if err != nil {
			log.Fatal(err)
		}
		results = string(bytes)
	}

	return results
}

func ReadFile(file *string) []string {
	var data []string

	fd, err := os.Open(*file)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}

	err = scanner.Err()
	if err != nil {
		log.Fatal(err)
	}

	return data
}

func FileWrite(data []byte, data_file string) {
	file, err := os.OpenFile(data_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("[!] Error opening output file:", err)
		return
	}
	defer file.Close()

	// Write the JSON data followed by a newline
	_, err = file.Write(data)
	if err != nil {
		log.Println("[!] Error writing JSON data to output file:", err)
	}

	_, err = file.WriteString("\n")
	if err != nil {
		log.Println("[!] Error writing newline to output file:", err)
	}
}

func CreateJson(response Result) []byte {
	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Println("[!] Error marshaling JSON:", err)
		return nil
	}

	return jsonData
}
