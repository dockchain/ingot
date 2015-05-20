package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"bufio"
	"net/url"
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
	"time"
)

var (
	rsaKey *rsa.PrivateKey
	rsaCertLoc = "sample.pem"
	EOFEvent = map[string]string{"Status": "EOF"}
	dockerURL = "unix:///var/run/docker.sock"
	parsedURL *url.URL
	pemFile = "sample.pem"
	caPem string
	caCert string
	caKey string
	dockerClient *docker.Client
	logTargetURL = "https://logs.dockcha.in/api/1/post-logs"
)

// initialize the global vars
func setMeUp() {
	var err error

	// define an alternative place for the RSA private key
	// used for signing the JSON blobs
	if certLoc := os.Getenv("INGOT_CERT_LOC") ; len(certLoc) != 0 {
		rsaCertLoc = certLoc
	}

	// get the file and decode it
	dat, err2 := ioutil.ReadFile(rsaCertLoc)
	
	if err2 != nil {
		panic(err)
	}
	
	block, _ := pem.Decode(dat)

	rsaKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)


	// Figure out the Docker Host... do we use the
	// default /var/run/docker.sock or something else?

	if url := os.Getenv("DOCKER_HOST"); len(url) != 0 {
		dockerURL = url
	}

	parsedURL, err = url.Parse(dockerURL)

	if err != nil {
		panic(err)
	}

	// If there's CERT path information, slurp it in
	if certPath := os.Getenv("DOCKER_CERT_PATH"); len(certPath) != 0 {
		caPem = fmt.Sprintf("%s/ca.pem", certPath)
		caCert = fmt.Sprintf("%s/cert.pem", certPath)
		caKey = fmt.Sprintf("%s/key.pem", certPath)
		dockerClient, err = docker.NewTLSClient(dockerURL, caCert,
			caKey, caPem)
	} else {
		dockerClient, err = docker.NewClient(dockerURL)
	}


	// if there was an error creating the client, bail
	if err != nil {
		panic(err)
	}

	// set up the target log host
	if url := os.Getenv("TARGET_LOG_HOST"); len(url) != 0 {
		logTargetURL = url
	}

}

func signBytes(bytes []byte) (s[]byte, err error) {
	hasher := sha256.New()
	hasher.Write( bytes)
	hash := hasher.Sum(nil)
	var h crypto.Hash
	
	return rsa.SignPKCS1v15(rand.Reader, rsaKey, h, hash)
}

func postInfo(info []interface{}) {
	fmt.Println("posting ", info)
}

func aggregate(incoming chan interface{}) {
	buffer := make([]interface{}, 0)
	timeout := time.After(time.Second * 5)

	sendIt := func () {
		if len(buffer) > 0 {
			go postInfo(buffer)
			buffer = make([]interface{}, 0)
		}
		timeout = time.After(time.Second * 5)
	}
	
	for {
		select {
		case evt := <- incoming:
			buffer = append(buffer, evt)
			if len(buffer) > 250 {
				sendIt()
			}

		case <- timeout:
			sendIt()
		}
	}
}

type inspect struct {
	name string
}

type history struct {
	name string
}



func imageFetcher(requests, aggregatorChan chan interface{}) {
	known := map[string]bool{}

	for req := range requests {
		switch r := req.(type) {
		case string:
			if !known[r] {
				known[r] = true
			}
			// dockerClient.
		case inspect:
			if !known[r.name] {
				known[r.name] = true
			}
		case history:
			if !known[r.name] {
				known[r.name] = true
			}
		}
	}
}

func processJSONMessages(msgChan, aggregateChan, imageHistoryListener chan interface{}) {
	for evt := range msgChan {
		aggregateChan <- evt // aggregate it

		// if it's a "create" message, then 
		// post the image information
		switch e := evt.(type) {
		case map[string]interface{}:
			switch e["status"].(string) {
			case "create":
				if from := e["from"].(string); len(from) != 0 {
					imageHistoryListener <- from
				}
			}
		}
	}
}

func main() {
	setMeUp()

	input := make(chan interface{})

	go func() {
		bio := bufio.NewReader(os.Stdin)
		line, _, _ := bio.ReadLine()
		input <- line
	}()

	// don't backup
	eventListener := make(chan interface{}, 1000)

	// never, ever backup
	imageHistoryListener := make(chan interface{}, 5000)
	
	errChan := make(chan error, 10)

	aggregatorChan := make(chan interface{}, 500)

	go func() {
		for evt := range errChan {
			fmt.Println(evt)
		}
	}()

	go aggregate(aggregatorChan)

	go imageFetcher(imageHistoryListener, aggregatorChan)

	go processJSONMessages(eventListener, aggregatorChan, 
		imageHistoryListener)

	// go func() {
	// 	for evt := range mapListener {
	// 		marshalled, _ := json.Marshal(evt)
	// 		signed, _ := signBytes(marshalled)
	// 		fmt.Println("Signed ", signed)

	// 		fmt.Println("encoded: ", string(marshalled[:]))
	// 	}
	// }()

	ehErr := eventHijack(0, eventListener, errChan)

	if ehErr != nil {
		panic(ehErr)
	}

	fmt.Println("Press 'enter' to stop")
	<- input
}

func eventHijack( startTime int64, eventChan chan interface{}, errChan chan error) error {
	client := dockerClient
	protocol := parsedURL.Scheme
	address := parsedURL.Path

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
		decoder.UseNumber()
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
