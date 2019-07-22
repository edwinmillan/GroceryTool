package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type credentials struct {
	Key    string
	Token  string
	CardID string
}

type requester struct {
	baseURL string
	creds   credentials
}

func (req requester) buildURL(components ...string) (url string) {
	base := []string{req.baseURL}
	auth := fmt.Sprintf("?key=%s&token=%s", req.creds.Key, req.creds.Token)
	components = append(components, auth)
	base = append(base, components...)

	url = strings.Join(base, "/")

	return
}

func (req requester) cardURL() string {
	return req.buildURL("cards", req.creds.CardID)

}

func (req requester) cardChecklistsURL() string {
	return req.buildURL("cards", req.creds.CardID, "checklists")
}

func (req requester) checkItemURL(checklistID, checkItemID string) string {
	return req.buildURL("checklists", checklistID, "checkItems", checkItemID)
}

func (req requester) getCheckLists() (jsonData []map[string]interface{}) {
	response := callAPI("GET", req.cardChecklistsURL())
	defer response.Body.Close()

	err := json.NewDecoder(response.Body).Decode(&jsonData)
	checkErr(err)

	return
}

func (req requester) cleanChecklists(checklistItems []map[string]string) {
	for _, entry := range checklistItems {
		for checklistID, checkItemID := range entry {
			checkItemURL := req.checkItemURL(checklistID, checkItemID)
			fmt.Printf("Deleting completed entry [Checklist: %s] [ItemID: %s]\n", checklistID, checkItemID)
			_ = callAPI("DELETE", checkItemURL)
		}
	}
}

func filterChecklists(checklistName string, checklists []map[string]interface{}) []interface{} {
	for _, checklist := range checklists {
		if checklist["name"] == checklistName {
			if checkItems, ok := checklist["checkItems"].([]interface{}); ok {
				return checkItems
			}
		}
	}

	return make([]interface{}, 0)
}

func completedCheckItems(checkItems []interface{}) (complete []map[string]string) {
	for _, checkItem := range checkItems {
		if checkProp, ok := checkItem.(map[string]interface{}); ok {
			if checkProp["state"] == "complete" {
				entry := map[string]string{
					checkProp["idChecklist"].(string): checkProp["id"].(string),
				}
				complete = append(complete, entry)
			}
		}
	}
	return
}

func callAPI(method, url string) (response *http.Response) {
	// Assume the connection wont be open longer than the timeout period set
	var httpClient = &http.Client{Timeout: 20 * time.Second}

	request, err := http.NewRequest(method, url, nil)
	checkErr(err)

	response, err = httpClient.Do(request)
	checkErr(err)

	return
}

func loadcredentials(fileName string) (creds credentials) {
	credFile, err := ioutil.ReadFile(fileName)
	checkErr(err)

	err = json.Unmarshal(credFile, &creds)
	checkErr(err)

	return
}

func newRequester(credFileName string) requester {
	req := requester{}
	req.baseURL = "https://api.trello.com/1"
	req.creds = loadcredentials(credFileName)

	return req
}

func main() {
	req := newRequester("credentials.json")

	rawChecklists := req.getCheckLists()
	checkItems := filterChecklists("Other", rawChecklists)
	req.cleanChecklists(completedCheckItems(checkItems))

}
