# go-cfgwatch
Go based Kubernetes app - Watches for ConfigMap changes and soft recycles the business logic


```
Purpose: 
Watching a loaded configuration map and on change trigger a soft restart of the business logic which is running in a Go routine. 

Info:
In this case the logic is a web server running in a Go Routine which is reading a value from memory and then outputting it when called.    

Takeaway:
The business logic is running in a Go routine and its execution and termination are handled by a simple supervisor routine in the main function that is triggered(stop/restart) by the file watcher function which is also running in its own Go routine. This approach is most effective with stateless applications. State full application would require additional logic to ensure all writes are complete prior to restart, however this should not be difficult to achieve. 

```


Rebuild
```
git clone repo
go mod tidy
env GOOS=linux GOARCH=amd64 go build -o app main.go
docker build -t go-cfgwatch .
docker tag go-cfgwatch <repo>/go-cfgwatch
docker push <repo>/go-cfgwatch
```

Run
```
adjust go-op-rc.yaml (update repo)

kubectl apply -f configmap.yaml -n default
kubectl apply -f go-op-rc.yaml -n default
```

Testing
```
Tail Pod Logs and/or expose via SVC and access with browser

Update Config Map

Watch for Change. 

```

Example Output
![Example Output](/screencap/screencap.png)

