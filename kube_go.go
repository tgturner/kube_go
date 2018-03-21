package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

const (
	MyDB = "kubedb"
)

func main() {
	timeDur, err := time.ParseDuration("60s")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://localhost:8086",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  MyDB,
		Precision: "m",
	})
	if err != nil {
		log.Fatal(err)
	}
	for true {
		command := "kubectl"
		args := []string{"top", "pods"}
		cmdOut, err := exec.Command(command, args...).Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		output := string(cmdOut)
		newOut := strings.Split(output, "\n")
		testFunc := func(s string) bool {
			return (strings.HasPrefix(s, "phoenix") && !strings.HasPrefix(s, "phoenix-prod-armailer"))
		}
		filteredOut := choose(newOut, testFunc)
		postEach(filteredOut, bp, c)
		time.Sleep(timeDur)
	}
}

func choose(ss []string, test func(string) bool) (ret []string) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func postEach(ss []string, bp client.BatchPoints, c client.Client) {
	re := regexp.MustCompile("[0-9]+")
	for _, s := range ss {
		item := strings.Fields(s)
		tags := map[string]string{"host": item[0]}
		mem, err := strconv.Atoi(re.FindAllString(item[2], -1)[0])
		if err != nil {
			log.Fatal(err)
		}
		cpu, err := strconv.Atoi(re.FindAllString(item[1], -1)[0])
		if err != nil {
			log.Fatal(err)
		}
		fieldsMem := map[string]interface{}{
			"memory": mem,
		}
		fieldsCPU := map[string]interface{}{
			"cpu": cpu,
		}
		ct, err := client.NewPoint("kube_mem", tags, fieldsMem, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		mt, err := client.NewPoint("kube_cpu", tags, fieldsCPU, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		bp.AddPoint(ct)
		bp.AddPoint(mt)
	}
	if err := c.Write(bp); err != nil {
		log.Fatal(err)
	}
}
