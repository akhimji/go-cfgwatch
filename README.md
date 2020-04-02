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

Example Output
![Example Output](/screencap/screencap.png)