package n26

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	n26API    = "https://api.tech26.de"
	basicAuth = "YW5kcm9pZDpzZWNyZXQ="
)

// N26 is a n26 client.
type N26 struct {
	token    string
	username string
	password string
}

// New returns a new n26 client given login information.
func New(username, password string) *N26 {
	return &N26{
		username: username,
		password: password,
	}
}

// login to n26 account.
func (n *N26) login() error {
	if n.token == "" {
		client := &http.Client{}
		data := url.Values{}
		data.Set("grant_type", "password")
		data.Set("username", n.username)
		data.Set("password", n.password)

		req, err := http.NewRequest(
			http.MethodPost,
			n26API+"/oauth/token",
			bytes.NewBufferString(data.Encode()),
		)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Basic "+basicAuth)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		out := struct {
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			Scope        string `json:"scope"`
		}{}

		fmt.Println(resp.StatusCode)

		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&out)
		if err != nil {
			return err
		}

		n.token = out.AccessToken
	}

	return nil
}

// Statements returns a table of statements in the defined period.
func (n *N26) Statements(from, to time.Time) ([][]string, error) {
	err := n.login()
	if err != nil {
		return nil, err
	}

	fromMili := from.UnixNano() / 1000000
	toMili := to.UnixNano() / 1000000

	client := &http.Client{}

	req, err := http.NewRequest(
		http.MethodGet,
		n26API+fmt.Sprintf("/api/smrt/reports/%d/%d/statements", fromMili, toMili),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "bearer "+n.token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	csvReader := csv.NewReader(resp.Body)
	return csvReader.ReadAll()
}
