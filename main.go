package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

const infobloxURL = "/wapi/v2.12.1/ipv4address?ip_address="

type Server struct {
	ipAddress string
	network   string
}

// GetFile is a function that gets access to a file based on the file name.
func GetFile(fileName string) (string, error) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	return string(file), nil
}

// GetConfig is a function that takes the contents of a file as a parameter as well as
// a pattern to use as a filter to return results as strings.
func GetConfig(file, pattern string) ([]string, error) {
	regexer, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	results := regexer.FindAllString(file, -1)
	return results, nil
}

func GetInfobloxNetwork(ipAddress, hostName string) (string, error) {
	// Create an http client
	client := &http.Client{}

	// Create a request
	req, err := http.NewRequest("GET", "https://"+hostName+infobloxURL+ipAddress, nil)
	if err != nil {
		return "", err
	}

	// Add headers to the request
	req.Header.Add("Authorization", os.Getenv("infobloxAuth"))

	// Send the request
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	// Defer the connection close
	defer res.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var infobloxResp []interface{}
	json.Unmarshal(body, &infobloxResp)

	// Parse the response for the network data
	for _, data := range infobloxResp {
		return data.(map[string]interface{})["network"].(string), nil
	}
	var infobloxError interface{}
	json.Unmarshal(body, &infobloxError)
	return infobloxError.(map[string]interface{})["text"].(string), nil
}

func GetServers(fileName, hostName string) ([]Server, error) {
	file, err := GetFile(fileName)
	if err != nil {
		return nil, err
	}
	ipAddresses, err := GetConfig(file, "\\b(?:[0-9]{1,3}\\.){3}[0-9]{1,3}\\b")
	if err != nil {
		return nil, err
	}
	var servers []Server
	for _, ipAddress := range ipAddresses {
		var server Server
		server.ipAddress = ipAddress
		server.network, err = GetInfobloxNetwork(ipAddress, hostName)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

// CreateFile is a function that accepts a file name as a parameter and returns a pointer to a file.
func CreateFile(fileName string) (*os.File, error) {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func main() {
	fileName := os.Args[1]
	infobloxHostname := os.Args[2]
	servers, err := GetServers(fileName, infobloxHostname)
	if err != nil {
		fmt.Println(err)
	}
	networkMap := make(map[string]string)
	for _, networkAddr := range servers {
		networkMap[networkAddr.network] = ""
	}
	file, err := CreateFile("networks.txt")
	if err != nil {
		fmt.Println(err)
	}
	for net, _ := range networkMap {
		fmt.Fprintln(file, net)
	}
}
