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
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
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

func watchfile(sendch chan<- string) {
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

	chnl := make(chan string)
	var msg string
	done := make(chan bool, 1)
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	server := newWebserver(logger)
	go runWeb(server, done)

	go watchfile(chnl)

	for {
		time.Sleep(1 * time.Second)
		msg = <-chnl
		if msg == "End" {
			fmt.Println("Ping")
			resp, err := http.Get("http://127.0.0.1:8080")
			if err != nil {
				// handle err
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(err)
				}
				bodyString := string(bodyBytes)
				log.Println(bodyString)
			}
		} else if msg == "Restart" {
			log.Println("Config Change.. Restarting in Main!")
			//os.Exit(0)
			fmt.Println("Attempting to Stop WebServer")
			time.Sleep(5 * time.Second)
			gracefullShutdown(server, logger, done)
			time.Sleep(5 * time.Second)
			fmt.Println("Attmepting To Restart")
			logger = log.New(os.Stdout, "http: ", log.LstdFlags)
			server = newWebserver(logger)
			go runWeb(server, done)
		}
	}

}
