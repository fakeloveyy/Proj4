package main

import (
    "kvpaxos"

    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
)

var data *kvpaxos.KVPaxosMap
var nodeId int
var peers []string
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
    peers = make([]string, len(config))

    for i := 0; i < len(peers); i++ {
        peer, exist := config[fmt.Sprintf("n%02d", i + 1)]
        if !exist {
            return errors.New("config file error")
        }
        peers[i] = peer
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
    if !(0 < nodeId && nodeId <= len(peers)) {
        return errors.New("node id range error")
    }
    return nil
}

func startServer() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot)
    mux.HandleFunc("/kv/insert", handleInsert)
    mux.HandleFunc("/kv/delete", handleDelete)
    mux.HandleFunc("/kv/get", handleGet)
    mux.HandleFunc("/kv/update", handleUpdate)
    mux.HandleFunc("/kvman/countkey", handleCountkey)
    mux.HandleFunc("/kvman/dump", handleDump)
    mux.HandleFunc("/kvman/shutdown", handleShutdown)
    http.ListenAndServe(port, mux)
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

    data = kvpaxos.NewKVPaxosMap(peers, nodeId - 1)

    startServer()
}

//-------------------------------------------------------//

// Handler functions

func handleRoot(wfile http.ResponseWriter, request *http.Request) {
    fmt.Fprintf(wfile,
`<html>
<body>
  <p>Insert a (key, value) pair</p>
  <form action="/kv/insert" method="post">
    <p>Key: <input type="text" name="key" /></p>
    <p>Value: <input type="text" name="value" /></p>
    <input type="submit" value="submit" />
  </form>

  <p>Delete a key</p>
  <form action="/kv/delete" method="post">
    <p>Key: <input type="text" name="key" /></p>
    <input type="submit" value="submit" />
  </form>

  <p>Get a value from key</p>
  <form action="/kv/get" method="get">
    <p>Key: <input type="text" name="key" /></p>
    <input type="submit" value="submit" />
  </form>

  <p>Update a (key, value) pair</p>
  <form action="/kv/update" method="post">
    <p>Key: <input type="text" name="key" /></p>
    <p>Value: <input type="text" name="value" /></p>
    <input type="submit" value="submit" />
  </form>

  <form action="/kvman/countkey" method="get">
    <input type="submit" value="countkey" />
  </form>

  <form action="/kvman/dump" method="get">
    <input type="submit" value="dump" />
  </form>

  <form action="/kvman/shutdown" method="get">
    <input type="submit" value="shutdown" />
  </form>
</body>
</html>`)
}

// Method: POST
// Arguments: key=k&value=v
// Return: {"success":"<true or false>"}
func handleInsert(wfile http.ResponseWriter, request *http.Request) {
    err := request.ParseForm()
    if err != nil {
        fmt.Fprintf(wfile, `{"success":"false"}`)
        return
    }

    keys, found_key := request.Form["key"]
    values, found_value := request.Form["value"]
    if !(found_key && found_value) {
        fmt.Fprintf(wfile, `{"success":"false"}`)
        return
    }

    key := keys[0]
    value := values[0]
    if (len(key) == 0 || len(value) == 0) {
        fmt.Fprintf(wfile, `{"success":"false"}`)
        return
    }

    ok := data.Put(key, value)
    if ok {
        fmt.Fprintf(wfile, `{"success":"true"}`)
    } else {
        fmt.Fprintf(wfile, `{"success":"false"}`)
    }
}

// Method: POST
// Arguments: key=k
// Return: {"success":"<true or false>","value":"<value deleted>"}
func handleDelete(wfile http.ResponseWriter, request *http.Request) {
    err := request.ParseForm()
    if err != nil {
        fmt.Fprintf(wfile, `{"success":"false","value":""}`)
        return
    }

    keys, found_key := request.Form["key"]
    if !found_key {
        fmt.Fprintf(wfile, `{"success":"false","value":""}`)
        return
    }

    key := keys[0]
    if len(key) == 0 {
        fmt.Fprintf(wfile, `{"success":"false","value":""}`)
        return
    }

    ok, value := data.Delete(key)
    if ok {
        fmt.Fprintf(wfile, `{"success":"true","value":"` + value + `"}`)
    } else {
        fmt.Fprintf(wfile, `{"success":"false","value":""}`)
    }
}

// Method: Get
// Arguments: key=k
// Return: {"success":"<true or false>","vaule":"<value>"}
func handleGet(wfile http.ResponseWriter, request *http.Request) {
    keys, found_key := request.URL.Query()["key"]
    if !found_key {
        fmt.Fprintf(wfile, `{"success":"false","value":""}`)
        return
    }

    key := keys[0]
    if len(key) == 0 {
        fmt.Fprintf(wfile, `{"success":"false","value":""}`)
        return
    }

    ok, value := data.Get(key)
    if ok {
        fmt.Fprintf(wfile, `{"success":"true","value":` + value + `"}`)
    } else {
        fmt.Fprintf(wfile, `{"success":"false","value":""}`)
    }
}

// Method: POST
// Arguments: key=k&value=v
// Return: {"success":"<true or false>"}
func handleUpdate(wfile http.ResponseWriter, request *http.Request) {
    err := request.ParseForm()
    if err != nil {
        fmt.Fprintf(wfile, `{"success":"false"}`)
        return
    }

    keys, found_key := request.Form["key"]
    values, found_value := request.Form["value"]
    if !(found_key && found_value) {
        fmt.Fprintf(wfile, `{"success":"false"}`)
        return
    }

    key := keys[0]
    value := values[0]
    if len(key) == 0 || len(value) == 0 {
        fmt.Fprintf(wfile, `{"success":"false"}`)
        return
    }

    ok := data.Update(key, value)
    if ok {
        fmt.Fprintf(wfile, `{"success":"true"}`)
    } else {
        fmt.Fprintf(wfile, `{"success":"false"}`)
    }
}

// Return {"result":"<number of keys>"}
func handleCountkey(wfile http.ResponseWriter, request *http.Request) {
    fmt.Fprintf(wfile, `{"result":"%d"}`, data.Count())
}

// Return [["<key>","<value>"], ...]
func handleDump(wfile http.ResponseWriter, request *http.Request) {
    fmt.Fprintf(wfile, data.Dump())
}

func handleShutdown(wfile http.ResponseWriter, request *http.Request) {
    data.Shutdown()
}
