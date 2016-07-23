package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

/*
 * If you get "Accept error: too many open files":
 * http://stackoverflow.com/a/880573
 */

type MiogType int

const (
	Mioga MiogType = iota
	Miogo
)

type TestingConfiguration struct {
	Email       string
	Password    string
	ServerURL   string
	RequestFile string
	Workers     int
	Duration    time.Duration
}

var client *http.Client

func main() {
	cookieJar, _ := cookiejar.New(nil)

	client = &http.Client{Jar: cookieJar}

	result := TestMiog(TestingConfiguration{
		Email:       "admin@miogo.tld",
		Password:    "ChangeMe",
		ServerURL:   "http://localhost:8070",
		RequestFile: "/hello.txt",
		Workers:     2,
		Duration:    10 * time.Second,
	}, Miogo)

	fmt.Printf("Requests for Miogo: %d\n", result)

	/*result = TestMiog(TestingConfiguration{
		Email:       "root@localhost.tld",
		Password:    "admin",
		ServerURL:   "http://localhost:8080",
		RequestFile: "/Mioga2/Mioga-FR/public/Administrateurs/hello.txt",
		Workers:     2,
		Duration:    10 * time.Second,
	}, Mioga)

	fmt.Printf("Requests for Mioga: %d\n", result)*/
}

func DoRequest(rtype, url, params string) {
	request, _ := http.NewRequest(rtype, url, strings.NewReader(params))

	if rtype == "POST" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	res, err := client.Do(request)

	if err != nil {
		panic(err.Error())
	}

	if res.StatusCode != 200 {
		panic(res.Status + " (" + url + ")")
	}
}

func TestMiog(conf TestingConfiguration, miog MiogType) (result int) {
	if miog == Mioga {
		DoRequest("POST", conf.ServerURL+"/Mioga2/login/LoginWS", "login="+conf.Email+"&password="+conf.Password)
	} else {
		DoRequest("POST", conf.ServerURL+"/Login", "email="+conf.Email+"&password="+conf.Password)
	}

	results := make(chan int)
	shutdown := make(chan bool)

	for i := 0; i < conf.Workers; i++ {
		go func() {
			req := 0

			for {
				select {
				case <-shutdown:
					results <- req
					return
				default:
					if miog == Mioga {
						DoRequest("GET", conf.ServerURL+conf.RequestFile, "")
					} else {
						DoRequest("POST", conf.ServerURL+"/GetFile", "path="+conf.RequestFile)
					}
					req++
				}
			}
		}()
	}

	go func() {
		time.Sleep(conf.Duration)

		for i := 0; i < conf.Workers; i++ {
			shutdown <- true
		}
	}()

	for i := 0; i < conf.Workers; i++ {
		// Wait for worker result
		result += <-results
	}

	return
}
