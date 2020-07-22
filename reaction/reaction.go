/*! \file reaction.go
    \brief Main file for the reaction service.
    Written in GO
    Created 2016-11-14 By Nathan Thomas
    
    The goal here is to make a simple util that can monitor an ip:port combo and execute a command if it fails

    do a 
    kill -10 pid
    to cause this to execute the next action
*/

package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
	"sync"
	"flag"
    "net"
    "encoding/json"
    "syscall"
    "os/exec"
    "strings"
    "bytes"
)

const APP_VER = "0.1"

  //-------------------------------------------------------------------------------------------------------------------------//
 //----- STRUCTS -----------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------------//

type config_t struct {
    Interval    int     `json:"interval"`
    Task   struct {
        IP      string  `json:"ip"`
        Port    int     `json:"port"`
        Init    bool    `json:"init"`
        Actions []string    `json:"actions"`
        CurrentIndex    int `json:"-"`
    }
    Running     bool    `json:"-"`
}

  //-------------------------------------------------------------------------------------------------------------------------//
 //----- PRIVATE FUNCTIONS -------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------------//

/*! \brief Reads in our config file
 */
func readConfig (loc string) (config config_t, err error) {
    //Read in the eggs
    configFile, err := os.Open(loc) //try the file
    
    if err == nil {
        defer configFile.Close()
        jsonParser := json.NewDecoder(configFile)
        err = jsonParser.Decode(&config)
        if err == nil {
            if len(config.Task.IP) < 4 { 
                err = fmt.Errorf("ip for the task appears invalid") 
            } else if config.Task.Port < 1 { 
                err = fmt.Errorf("port for the task appears invalid") 
            } else if len(config.Task.Actions) < 1 { 
                err = fmt.Errorf("Task has no actions") 
            }
        }
    } else {
        err = fmt.Errorf("Unable to open '%s' file :: " + err.Error(), loc)
    }
    return
}

func action (config *config_t) {
    config.Running = true   //flag so we don't do this again until we're ready
    config.Task.CurrentIndex++
    if config.Task.CurrentIndex >= len(config.Task.Actions) { config.Task.CurrentIndex = 0 }    //reset the index

    fmt.Printf("Task error for %s:%d\nExecuting action %s\n", config.Task.IP, config.Task.Port, config.Task.Actions[config.Task.CurrentIndex])

    args := strings.Fields(config.Task.Actions[config.Task.CurrentIndex])
    cmd := exec.Command (args[0], args[1:]...)
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println(out.String())
    config.Running = false  //reset this when we exit this function
}

func urlCheck (config *config_t) {
    if config.Running { return }    //we're trying to run the action, which can take time, so ignore this until we finish the last call

    //check the ip:port
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.Task.IP, config.Task.Port))
    if err == nil {
        defer conn.Close()
    } else {    //this is bad, means it didn't work
        action (config)
    }
}

  //-------------------------------------------------------------------------------------------------------------------------//
 //----- MAIN --------------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------------//

func main() {
	//handle any passed in flags
	configFileFlag := flag.String("c", "reaction.json", "Location of config file")
	versionFlag := flag.Bool("v", false, "Returns the version")
	flag.Parse()
    defer fmt.Printf("\nFor every reaction there's an equal and opposite reaction\n\n")
	
	if *versionFlag {
		fmt.Printf("\nReaction Version: %s\n\n", APP_VER)
		os.Exit(0)
	}
	
	config, err := readConfig(*configFileFlag)
	if err != nil {	//see if we initalized correctly
		fmt.Println(err)
		os.Exit(0)
	}
	
	//check the flags
	if config.Interval < 1 { config.Interval = 10 }    //default to 10 seconds
	
	wg := new(sync.WaitGroup)
	wg.Add(1)
	
	//this handles killing the service gracefully
	c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

    go func(wg *sync.WaitGroup){
		<-c
		//for sig := range c {
			// sig is a ^C, handle it
			fmt.Println("Reaction service exiting gracefully")
			//os.Exit(0)
			defer wg.Done()
		//}
	}(wg)

    //this handles intentionally executing the next action
    switchSignal := make(chan os.Signal, 1)
    signal.Notify(switchSignal, syscall.SIGUSR1)

    go func() {
        for true {
            <-switchSignal
            fmt.Println("Next action due to signal")
            action (&config)
        }
    }()
	
	//this is our polling interval
	ticker := time.NewTicker(time.Second * time.Duration(config.Interval))	//check every interval
	go func() {
		for range ticker.C {
			urlCheck(&config)    //do our check
		}
	} ()
	
	urlCheck(&config)    //do our check
	
	wg.Wait()	//wait for the subordinate and possible main to finish
}
