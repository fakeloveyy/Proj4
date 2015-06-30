package kvclient

import (
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "strings"
)

type KVClient struct {
    ip []string
}

func NewKVClient() (*KVClient, error) {
    kvclient := &KVClient{}

    bytes, err := ioutil.ReadFile("conf/settings.conf")
    if err != nil {
        return nil, err
    }

    var config map[string]string
    err = json.Unmarshal(bytes, &config)
    if err != nil {
        return nil, err
    }

    port, exist := config["port"]
    if !exist {
        return nil, errors.New("config file error")
    }
    port = ":" + port

    delete(config, "port")
    if (len(config) >= 100) {
        return nil, errors.New("config file error")
    }
    kvclient.ip = make([]string, len(config))

    for i := 0; i < len(kvclient.ip); i++ {
        peer, exist := config[fmt.Sprintf("n%02d", i + 1)]
        if !exist {
            return nil, errors.New("config file error")
        }
        kvclient.ip[i] = strings.Split(peer, ":")[0] + port
    }
    return kvclient, nil
}

func (kvclient *KVClient) Insert(key string, value string) ([]byte, error) {
    for i := 0; i < len(kvclient.ip); i++ {
        resp, err := http.PostForm("http://" + kvclient.ip[i] + "/kv/insert", url.Values{"key":{key}, "value":{value}})
        if err != nil {
            return make([]byte, 0), err
        }
        defer resp.Body.Close()
        return ioutil.ReadAll(resp.Body)
    }
    return make([]byte, 0), errors.New("all servers failed")
}

func (kvclient *KVClient) Get(key string) ([]byte, error) {
    for i := 0; i < len(kvclient.ip); i++ {
        resp, err := http.Get("http://" + kvclient.ip[i] + "/kv/get?key=" + key)
        if err != nil {
            return make([]byte, 0), err
        }
        defer resp.Body.Close()
        return ioutil.ReadAll(resp.Body)
    }
    return make([]byte, 0), errors.New("all servers failed")
}

func (kvclient *KVClient) Delete(key string) ([]byte, error) {
    for i := 0; i < len(kvclient.ip); i++ {
        resp, err := http.PostForm("http://" + kvclient.ip[i] + "/kv/delete", url.Values{"key":{key}})
        if err != nil {
            return make([]byte, 0), err
        }
        defer resp.Body.Close()
        return ioutil.ReadAll(resp.Body)
    }
    return make([]byte, 0), errors.New("all servers failed")
}

func (kvclient *KVClient) Update(key string, value string) ([]byte, error) {
    for i := 0; i < len(kvclient.ip); i++ {
        resp, err := http.PostForm("http://" + kvclient.ip[i] + "/kv/update", url.Values{"key":{key}, "value":{value}})
        if err != nil {
            return make([]byte, 0), err
        }
	    defer resp.Body.Close()
        return ioutil.ReadAll(resp.Body)
    }
    return make([]byte, 0), errors.New("all servers failed")
}

func (kvclient *KVClient) Countkey() ([]byte, error) {
    for i := 0; i < len(kvclient.ip); i++ {
        resp, err := http.Get("http://" + kvclient.ip[i] + "/kvman/countkey")
        if err != nil {
            return make([]byte, 0), err
        }
	    defer resp.Body.Close()
        return ioutil.ReadAll(resp.Body)
    }
    return make([]byte, 0), errors.New("all servers failed")
}

func (kvclient *KVClient) Dump() ([]byte, error) {
    for i := 0; i < len(kvclient.ip); i++ {
        resp, err := http.Get("http://" + kvclient.ip[i] + "/kvman/dump")
        if err != nil {
            return make([]byte, 0), err
        }
	    defer resp.Body.Close()
        return ioutil.ReadAll(resp.Body)
    }
    return make([]byte, 0), errors.New("all servers failed")
}