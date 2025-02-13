package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

var httpMethods = []string{
	"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "TRACE", "CONNECT",
	"PROPFIND", "PROPPATCH", "MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK", "CHECKOUT", 
	"MERGE", "REPORT", "SEARCH", "PURGE", "M-SEARCH", "NOTIFY", "SUBSCRIBE", "UNSUBSCRIBE",
}

type Result struct {
	URL     string `json:"url"`
	Method  string `json:"method"`
	Status  int    `json:"status"`
	Length  int    `json:"length"`
}

func testMethod(url, method string) (Result, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return Result{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return Result{URL: url, Method: method, Status: resp.StatusCode, Length: len(body)}, nil
}

func testSingleTarget(url string) {
	fmt.Println("[*] Testen van HTTP-methodes op:", url)
	var wg sync.WaitGroup
	for _, method := range httpMethods {
		wg.Add(1)
		go func(m string) {
			defer wg.Done()
			res, err := testMethod(url, m)
			if err == nil {
				fmt.Printf("[%s] %d (%d bytes)\n", res.Method, res.Status, res.Length)
			}
		}(method)
	}
	wg.Wait()
}

func testMultipleTargets(urls []string, method string, outputFile string) {
	fmt.Println("[*] Testen van", method, "op meerdere doelen...")
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			res, err := testMethod(u, method)
			if err == nil && res.Status != 405 {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
		}(url)
	}
	wg.Wait()

	data, _ := json.MarshalIndent(results, "", "  ")
	_ = ioutil.WriteFile(outputFile, data, 0644)
	fmt.Println("[+] Resultaten opgeslagen in", outputFile)
}

func testMultipleTargetsMultipleMethods(urls []string, outputFile string) {
	fmt.Println("[*] Testen van meerdere HTTP-methodes op meerdere doelen...")
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, url := range urls {
		for _, method := range httpMethods {
			wg.Add(1)
			go func(u, m string) {
				defer wg.Done()
				res, err := testMethod(u, m)
				if err == nil && res.Status != 405 {
					mu.Lock()
					results = append(results, res)
					mu.Unlock()
				}
			}(url, method)
		}
	}
	wg.Wait()

	data, _ := json.MarshalIndent(results, "", "  ")
	_ = ioutil.WriteFile(outputFile, data, 0644)
	fmt.Println("[+] Resultaten opgeslagen in", outputFile)
}

func main() {
	singleTarget := flag.String("u", "", "Test meerdere HTTP-methodes tegen een enkele URL")
	urlListFile := flag.String("w", "", "Bestand met URLs om te testen")
	method := flag.String("method", "GET", "HTTP-methode om te testen (default: GET)")
	outputFile := flag.String("o", "results.json", "Outputbestand voor JSON-resultaten")
	combine := flag.Bool("combine", false, "Test meerdere methodes op meerdere URLs")

	flag.Parse()

	if *singleTarget != "" {
		testSingleTarget(*singleTarget)
	} else if *urlListFile != "" {
		data, err := ioutil.ReadFile(*urlListFile)
		if err != nil {
			fmt.Println("[-] Fout bij het lezen van het bestand:", err)
			return
		}
		urls := strings.Split(string(data), "\n")
		if *combine {
			testMultipleTargetsMultipleMethods(urls, *outputFile)
		} else {
			testMultipleTargets(urls, *method, *outputFile)
		}
	} else {
		fmt.Println("Gebruik:")
		fmt.Println("  -u <url>        Test alle HTTP-methodes op een enkele URL")
		fmt.Println("  -w <bestand>    Test HTTP-methode(s) op een lijst met URLs")
		fmt.Println("  -method <str>   HTTP-methode om te testen (default: GET)")
		fmt.Println("  -o <bestand>    JSON outputbestand (default: results.json)")
		fmt.Println("  -combine        Test meerdere methodes op meerdere URLs")
	}
}
