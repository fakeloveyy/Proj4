package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "strings"
)

var nodeId int
var ip []string
var port string

// Load configuration from "conf/settings.conf"
// Get all replicas' ip and port
func loadConfig() error {
    bytes, err := ioutil.ReadFile("conf/settings.conf")
    if err != nil {
        return err
    }

    var config map[string]string
    err = json.Unmarshal(bytes, &config)
    if err != nil {
        return err
    }

    var exist bool
    port, exist = config["port"]
    if !exist {
        return errors.New("config file error")
    }
    port = ":" + port

    delete(config, "port")
    if (len(config) >= 100) {
        return errors.New("config file error")
    }
    ip = make([]string, len(config))

    for i := 0; i < len(ip); i++ {
        peer, exist := config[fmt.Sprintf("n%02d", i + 1)]
        if !exist {
            return errors.New("config file error")
        }
        ip[i] = strings.Split(peer, ":")[0]
    }
    return nil
}

// Analyse the command-line arguments and get the node id
func loadNodeId() error {
    if (len(os.Args) != 2) {
        return errors.New("command line arguments error")
    }
    if n, err := fmt.Sscanf(os.Args[1], "n%d", &nodeId); n != 1 || err != nil {
        return errors.New("command line arguments error")
    }
    if !(0 < nodeId && nodeId <= len(ip)) {
        return errors.New("node id range error")
    }
    return nil
}

func main() {
    err := loadConfig()
    if err != nil {
        fmt.Println(err)
        return
    }

    err = loadNodeId()
    if err != nil {
        fmt.Println(err)
        return
    }

    http.Get("http://" + ip[nodeId] + "/kvman/shutdown")
}
