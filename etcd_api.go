/*
* @Author: Jim Weber
* @Date:   2016-05-18 22:10:02
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-09-27 23:18:50
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

func nameForNextInstance(unit string) string {
	rx := regexp.MustCompile("(.*)-([0-9.]+(-SNAPSHOT)?)")

	log.Println(unit)
	nameParts := rx.FindStringSubmatch(unit)

	appName := nameParts[1]
	return appName
}

func getAppVersionNumber(unit string) string {
	
	rx := regexp.MustCompile("(.*)-([0-9.]+(-SNAPSHOT)?)")
	// containerData := make(map[string]string)

	log.Println(unit)
	nameParts := rx.FindStringSubmatch(unit)

	appVersion := nameParts[2]
	return appVersion
}

func getNextInstance(host string, appName string) int64 {
	// TODO: host needs to be etcd host create cli arg for that
	// url := "http://" + host + "/v2/keys/nextinstance/" + appName
	url := "http://coreos.dev.crosschx.com:4001/v2/keys/nextinstance/" + appName // temp
	var curInstance int64
	var nextInstanceNum int64
	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("%s", err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
		}

		var etcdResp map[string]interface{}
		if err := json.Unmarshal(contents, &etcdResp); err != nil {
			panic(err)
		}

		if etcdResp["errorCode"] != nil {
			fmt.Println("Instance number does not exist. Initializing new key")
			// initialize instance key
			setInstanceNumber(host, appName, 10, 0)
			nextInstanceNum = 10
		} else {
			nodeResp := etcdResp["node"].(map[string]interface{})
			instanceValue := nodeResp["value"].(string)
			curInstance, _ = strconv.ParseInt(instanceValue, 0, 0)
			nextInstanceNum = curInstance + 1
			if nextInstanceNum >= 99 {
				// we are purposely not going over 99
				// always reset to 10 if we are at 99 or greater because of a bug
				nextInstanceNum = 10
			}
			setInstanceNumber(host, appName, nextInstanceNum, curInstance)
		}

	}

	return nextInstanceNum
}

func setInstanceNumber(host string, appName string, instanceNumber int64, prevValue int64) {
	// url := "http://" + host
	url := "http://coreos.dev.crosschx.com" //temp
	cfg := client.Config{
		Endpoints: []string{url + ":4001"},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	etcdClient, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	kapi := client.NewKeysAPI(etcdClient)
	setOptions := client.SetOptions{}
	if prevValue != 0 {
		setOptions.PrevValue = fmt.Sprintf("%d", prevValue)
	}
	_, err = kapi.Set(context.Background(), "/nextinstance/"+appName, fmt.Sprintf("%d", instanceNumber), &setOptions)
	if err != nil {
		log.Fatal(err)
	}
}

func instanceUp(host, appName, appVersion, instanceNumber string, waitSecs int) bool {
	var up bool
	// etcdURL := "http://" + host
	etcdURL := "http://coreos.dev.crosschx.com" //temp
	cfg := client.Config{
		Endpoints: []string{etcdURL + ":4001"},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	etcdClient, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	kapi := client.NewKeysAPI(etcdClient)
	watchOptions := client.WatcherOptions{0, false}
	watcher := kapi.Watcher("/services/instances/"+appName+"/"+appVersion+"@"+instanceNumber, &watchOptions)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(waitSecs)*time.Second)
	defer cancel()
	fmt.Println("Waiting", waitSecs, "seconds for container to be up")

	// start watching the fleet state of our unit / instance
	stateChan := make(chan string)
	quit := make(chan bool)
	go watchFleetState(host, appName, appVersion, instanceNumber, stateChan, quit)
	go func() {
		for {
			select {
			case state := <-stateChan:
				if state == "failed" {
					log.Println("Unit State", state)
					fmt.Println("Fleet unit entered failed state")
					quit <- true
					cancel()
					return
				}

				log.Println("Unit State", state)

			}

		}
	}()

	_, err = watcher.Next(ctx)
	if err != nil {
		if err == context.Canceled {
			// ctx is canceled by another routine
		} else if err == context.DeadlineExceeded {
			up = false
		} else {
			// handle error
			up = false
		}
	} else {
		up = true
		quit <- true
	}

	return up
}
