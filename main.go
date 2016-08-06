package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"

	"github.com/mikkeloscar/autobank/bank"
	"github.com/mikkeloscar/autobank/bank/db"
	"github.com/mikkeloscar/autobank/bank/n26"
)

func main() {
	conf, err := loadConfig("config.yml")
	if err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

	// initialize bank APIs
	banks := make(map[string]bank.Bank)

	if conf.Banks.N26 != nil {
		banks["n26"] = n26.New(
			conf.Banks.N26.User,
			conf.Banks.N26.Password,
		)
	}

	if conf.Banks.DB != nil {
		banks["db"] = db.New(
			conf.Banks.DB.Branch,
			conf.Banks.DB.Account,
			conf.Banks.DB.PIN,
		)
	}

	t1, err := time.Parse(time.RFC3339, "2016-01-01T00:00:00+00:00")
	if err != nil {
		panic(err)
	}
	t2 := time.Now().UTC()

	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/sheets.googleapis.com-go-quickstart.json
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}

	for bank, api := range banks {
		statements, err := api.Statements(t1, t2)
		if err != nil {
			log.Fatalf("failed to get statements for: %s - %s", bank, err)
		}

		sheetRange := fmt.Sprintf("%s!A1", bank)

		valRange := &sheets.ValueRange{
			MajorDimension: "ROWS",
			Range:          sheetRange,
			Values:         strTableToInterfaceTable(statements),
		}
		_, err = srv.Spreadsheets.Values.Update(conf.SpreadSheetID, sheetRange, valRange).
			ValueInputOption("USER_ENTERED").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from sheet. %v", err)
		}
	}
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("sheets.googleapis.com-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// strTableToInterfaceTable transforms a string table to an interface table
// required for the google sheets API.
func strTableToInterfaceTable(table [][]string) [][]interface{} {
	newTable := make([][]interface{}, 0, len(table))
	for _, row := range table {
		newRow := make([]interface{}, 0, len(row))
		for _, item := range row {
			newRow = append(newRow, item)
		}
		newTable = append(newTable, newRow)
	}

	return newTable
}
