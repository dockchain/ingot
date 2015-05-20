package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"bufio"
	"os"
	"net"
	"net/http"
	"crypto/x509"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/pem"
	"crypto"
	"io/ioutil"
	"net/http/httputil"
)

var (
	EOFEvent = map[string]string{"Status": "EOF"}
)

// recursively walk the JSONish data and find
// all float64 and float32 elements that can be represented
// as int64 and convert them
func fixFloat64(info interface{}) interface{} {
	switch v := info.(type) {
	case map[string]interface{}:
		ret := map[string]interface{}{}
		for k,tv := range v {
			ret[k] = fixFloat64(tv)
		}
		return ret
	case []interface{}:
		ret := make([]interface{}, len(v))
		for k,tv := range v {
			ret[k] = fixFloat64(tv)
		}
		return ret
	case float64:
		if v == float64(int64(v)) {
			return int64(v)
		} else {
			return v
		}
	case float32:
		if v == float32(int64(v)) {
			return int64(v)
		} else {
			return v
		}
	}
	return info
}

func signBytes(bytes []byte, key *rsa.PrivateKey) (s[]byte, err error) {
	hasher := sha256.New()
	hasher.Write( bytes)
	hash := hasher.Sum(nil)
	var h crypto.Hash
	
	return rsa.SignPKCS1v15(rand.Reader, key, h, hash)
}

func main() {
	dat, _ := ioutil.ReadFile("sample.pem")
	
	block, _ := pem.Decode(dat)

	key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	fmt.Println("Key ", key)

	input := make(chan interface{})

	go func() {
		bio := bufio.NewReader(os.Stdin)
		line, _, _ := bio.ReadLine()
		input <- line
	}()

	// FIXME deal with the endpoint
	endpoint := "unix:///var/run/docker.sock"
	
	client, _ := docker.NewClient(endpoint)

	mapListener := make(chan interface{}, 10)
	
	errChan := make(chan error, 10)

	go func() {
		for evt := range errChan {
			fmt.Println(evt)
		}
	}()


	go func() {
		for evt := range mapListener {
			evt = fixFloat64([]interface{}{evt})
			// switch t := evt.(type) {
			// case map[string]interface{}:
			// 	if tmp, v := t["time"].(float64); v {
			// 		fmt.Println("Are they eq?", tmp == float64(int64(tmp)))
			// 		t["time"] = int64(tmp)
			// 	}
			// 	switch v := t["time"].(type) {
			// 	case int:
			// 		fmt.Println("V is an int", v)
			// 	case float64:
			// 		fmt.Println("V is ", int(v))
			// 	default:
			// 		fmt.Println("Type", reflect.TypeOf(v))
			// 	}
			// 	fmt.Println("Time type: ", t["time"])
			// 	fmt.Println("yak type: ", t["yak"])
			// }

			marshalled, _ := json.Marshal(evt)
			signed, _ := signBytes(marshalled, key)
			fmt.Println("Signed ", signed)

			fmt.Println("encoded: ", string(marshalled[:]))
		}
	}()

	eventHijack("unix", "/var/run/docker.sock", 
		client, 0, mapListener, errChan)

	listener := make(chan *docker.APIEvents, 10)

	go func() {
		for evt := range listener {
			fmt.Println(evt)
		}
	}()

	client.AddEventListener(listener)

	fmt.Println("Press 'enter' to stop")
	<- input
}

func eventHijack(protocol string, address string, client *docker.Client, startTime int64, eventChan chan interface{}, errChan chan error) error {
	tlsConfig := client.TLSConfig

	uri := "/events"
	if startTime != 0 {
		uri += fmt.Sprintf("?since=%d", startTime)
	}
	if protocol != "unix" {
		protocol = "tcp"
	}
	var dial net.Conn
	var err error
	if tlsConfig == nil {
		dial, err = net.Dial(protocol, address)
	} else {
		dial, err = tls.Dial(protocol, address, tlsConfig)
	}
	if err != nil {
		return err
	}
	conn := httputil.NewClientConn(dial, nil)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}
	res, err := conn.Do(req)
	if err != nil {
		return err
	}
	go func(res *http.Response, conn *httputil.ClientConn) {
		defer conn.Close()
		defer res.Body.Close()
		decoder := json.NewDecoder(res.Body)
		for {
			var event interface{}
			if err = decoder.Decode(&event); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {					
					eventChan <- EOFEvent
					break
				}
				errChan <- err
			}
			eventChan <- event
		}
	}(res, conn)
	return nil
}
