package db

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL       = "https://meine.deutsche-bank.de"
	loginPagePath = "/trxm/db/"
	loginPath     = "/trxm/db/gvo/login/login.do"

	pageDisplayTransactions = "DisplayTransactions"
)

// DeutscheBank defines a client for getting statements from a Deutsche Bank
// account.
type DeutscheBank struct {
	branch    string
	account   string
	pin       string
	cookieJar *cookiejar.Jar
	client    *http.Client
}

// New returns a new DeutscheBank client given login information.
func New(branch, account, pin string) *DeutscheBank {
	return &DeutscheBank{
		branch:  branch,
		account: account,
		pin:     pin,
	}
}

// login logs into deutsche bank to the given page.
func (d *DeutscheBank) login(page string) (*http.Response, error) {
	var err error
	d.cookieJar, err = cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	d.client = &http.Client{
		Jar: d.cookieJar,
	}

	req, err := http.NewRequest("GET", baseURL+loginPagePath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Language", "en-US,en;q=0.8,da;q=0.6")
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	v := url.Values{}
	v.Set("gvo", "DisplayFinancialOverview")
	v.Add("loginTab", "pin")
	v.Add("process", "")
	v.Add("wknOrIsin", "")
	v.Add("quantity", "")
	v.Add("fingerprintToken", "")
	v.Add("fingerprintTokenVersion", "")
	v.Add("updateFingerprintToken", "false")
	v.Add("javascriptEnabled", "false")
	v.Add("branch", d.branch)
	v.Add("account", d.account)
	v.Add("subaccount", "00")
	v.Add("pin", d.pin)
	v.Add("quickLink", page)

	req, err = http.NewRequest("POST", baseURL+loginPath, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = d.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Statements returns a table of statements in the given period.
func (d *DeutscheBank) Statements(from, to time.Time) ([][]string, error) {
	resp, err := d.login(pageDisplayTransactions)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	// get refresh transaction date range path with special session value
	var path string
	var ok bool
	doc.Find("#accountTurnoversForm").Each(func(i int, s *goquery.Selection) {
		path, ok = s.Attr("action")
	})
	if !ok {
		return nil, fmt.Errorf("failed to find refresh form on page")
	}

	v := url.Values{}
	v.Set("subaccountAndCurrency", "00")
	v.Add("period", "dynamicRange")
	v.Add("periodStartMonth", fmt.Sprintf("%02d", from.Month()))
	v.Add("periodStartDay", fmt.Sprintf("%02d", from.Day()))
	v.Add("periodStartYear", fmt.Sprintf("%d", from.Year()))
	v.Add("periodEndMonth", fmt.Sprintf("%02d", to.Month()))
	v.Add("periodEndDay", fmt.Sprintf("%02d", to.Day()))
	v.Add("periodEndYear", fmt.Sprintf("%02d", to.Year()))
	v.Add("periodDays", "180")
	v.Add("searchString", "")

	req, err := http.NewRequest("POST", baseURL+path, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = d.client.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err = goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	// get refresh transaction date range path with special session value
	// var path string
	doc.Find(".csv a").Each(func(i int, s *goquery.Selection) {
		path, ok = s.Attr("href")
	})
	if !ok {
		return nil, fmt.Errorf("failed to find csv link on page")
	}

	req, err = http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	resp, err = d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	csvReader := csv.NewReader(resp.Body)
	csvReader.Comma = ';'
	csvReader.FieldsPerRecord = -1
	return csvReader.ReadAll()
}
