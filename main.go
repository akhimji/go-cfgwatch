/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ghodss/yaml"
)

const configFile = "/etc/config/configmap.yaml"

type configMap struct {
	Message string `yaml:"message"`
}

func loadConfig(configFile string) *configMap {
	conf := &configMap{}
	configData, _ := ioutil.ReadFile(configFile)
	_ = yaml.Unmarshal(configData, conf)

	return conf
}

func watchFile(sendch chan<- string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	err = watcher.Add(configFile)
	if err != nil {
		log.Fatal(err)
	}
	for {
		sendch <- "Ping"
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("file event:", event)
			sendch <- event.Name
			if event.Op == fsnotify.Write || event.Op == fsnotify.Remove {
				log.Println("modified file:", event.Name)
				sendch <- "Restart"
				watcher.Add(configFile)
			}
		default:
			sendch <- "End"
		}
	}
}

func gracefullShutdown(server *http.Server, logger *log.Logger, done chan<- bool) {

	fmt.Println("Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Could not gracefully shutdown the server: %v\n", err)
	}
	done <- true
}

func newWebserver(logger *log.Logger) *http.Server {
	conf := loadConfig(configFile)
	fmt.Println("Loading Config Map...")
	fmt.Println("Config Map Value: ", conf.Message)
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(conf.Message))
	})

	return &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}

func runWeb(server *http.Server, done <-chan bool) {
	fmt.Println("Server is ready to handle requests at", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
	}
	<-done
	fmt.Println("Server stopped")
}

func main() {

	filechnl := make(chan string)
	var msg string
	done := make(chan bool, 1)
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	server := newWebserver(logger)
	go runWeb(server, done)

	go watchFile(filechnl)

	for {
		time.Sleep(1 * time.Second)
		msg = <-filechnl
		if msg == "End" {
			fmt.Println("Polling Server..")
			resp, err := http.Get("http://127.0.0.1:8080")
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(err)
				}
				bodyString := string(bodyBytes)
				log.Println(bodyString)
				time.Sleep(2 * time.Second)
			}
		} else if msg == "Restart" {
			log.Println("Config Change...Restarting in Main Func...")
			//os.Exit(0)
			fmt.Println("Attempting to Stop WebServer...")
			gracefullShutdown(server, logger, done)
			time.Sleep(5 * time.Second)
			fmt.Println("Attmepting To Restart")
			logger = log.New(os.Stdout, "http: ", log.LstdFlags)
			server = newWebserver(logger)
			go runWeb(server, done)
		}
	}

}
