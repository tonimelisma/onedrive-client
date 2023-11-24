package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"github.com/tonimelisma/onedrive-sdk-go"
	"golang.org/x/oauth2"
)

// UTILITIES

func debug(config Configuration, message string) {
	if config.Debug {
		log.Println(message)
	}
}

// CONFIGURATION HANDLING

const configDir = ".onedrive-client"
const configFile = "config.json"

type Configuration struct {
	Token oauth2.Token `json:"token"`
	Debug bool         `json:"debug"`
}

func loadConfiguration() (thisConfig Configuration, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return thisConfig, fmt.Errorf("getting home directory: %v", err)
	}

	configPath := filepath.Join(homeDir, configDir, configFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return thisConfig, err
	}

	fileHandle, err := os.ReadFile(configPath)
	if err != nil {
		return thisConfig, fmt.Errorf("reading file: %v", err)
	}

	err = json.Unmarshal([]byte(fileHandle), &thisConfig)
	if err != nil {
		return thisConfig, fmt.Errorf("unmarshalling json: %v", err)
	}

	return thisConfig, nil
}

func saveConfiguration(thisConfig Configuration) (err error) {
	jsonData, err := json.MarshalIndent(thisConfig, "", "")
	if err != nil {
		return fmt.Errorf("marshalling json: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %v", err)
	}

	configDirPath := filepath.Join(homeDir, configDir)
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		err = os.Mkdir(configDirPath, 0700)
		if err != nil {
			return fmt.Errorf("creating configuration directory: %v", err)
		}
	}

	configFilePath := filepath.Join(configDirPath, configFile)
	err = os.WriteFile(configFilePath, jsonData, 0600)
	if err != nil {
		return fmt.Errorf("writing configuration: %v", err)
	}

	return nil
}

// MAIN PROGRAM

func main() {
	// Load configuration, and initialize OAuth2 tokens
	config, err := loadConfiguration()
	config.Debug = true
	if os.IsNotExist(err) {
		fmt.Println("No configuration file found.")
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't load configuration: %v\n", err)
		os.Exit(1)
	}

	var client *http.Client

	if config.Token.AccessToken == "" {
		debug(config, "No access token found.")
		client, err = authenticate(&config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "couldn't authenticate: %v\n", err)
			os.Exit(1)
		}
	} else {
		debug(config, "Access token found.")
		client, err = validateToken(&config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't validate token: %v\n", err)
			os.Exit(1)
		}
	}

	// OAuth2 tokens ready
	fmt.Println("Getting drives...")

	onedrive.GetMyDrives(client)
}

// OAUTH2 STUFF

func getOauth2Config() (context.Context, *oauth2.Config) {
	return context.Background(), &oauth2.Config{
		ClientID: "71ae7ad2-0207-4618-90d3-d21db38f9f7a",
		Scopes:   []string{"offline_access", "files.readwrite.all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
	}
}

func authenticate(config *Configuration) (*http.Client, error) {
	ctx, oauthConfig := getOauth2Config()

	codeVerifier, err := cv.CreateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("creating code verifier: %v", err)
	}
	codeChallenge := codeVerifier.CodeChallengeS256()

	codeChallengeAuthParam := oauth2.SetAuthURLParam("code_challenge", codeChallenge)
	codeChallengeMethodAuthParam := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	fmt.Println("Visit the following URL in your browser and authorize the app.")
	fmt.Println(oauthConfig.AuthCodeURL("local", codeChallengeAuthParam, codeChallengeMethodAuthParam))
	fmt.Println("")
	fmt.Println("You'll be redirected to an empty page. Copy-paste its URL here:")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("scanning [%v]: %v", code, err)
	}
	fmt.Println("")

	// Parse oauth callback URL to ensure there's no error
	parsedUrl, err := url.Parse(code)
	if err != nil {
		return nil, fmt.Errorf("parsing url [%v]: %v", code, err)
	}

	// Parse oauth callback URL's query parameters
	parsedQuery, err := url.ParseQuery(parsedUrl.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("parsing query [%v]: %v", code, err)
	}

	// If callback returned an error, bring it up
	if parsedQuery.Has("error") {
		if parsedQuery.Has("error_description") {
			return nil, fmt.Errorf("oauth authorization failed: %v: %v", parsedQuery.Get("error"), parsedQuery.Get("error_description"))
		} else {
			return nil, fmt.Errorf("oauth authorization failed: %v", parsedQuery.Get("error"))
		}
	}

	if !parsedQuery.Has("code") {
		return nil, fmt.Errorf("couldn't parse code in callback: %v", parsedQuery.Encode())
	}

	codeVerifierAuthParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
	token, err := oauthConfig.Exchange(ctx, parsedQuery.Get("code"), codeVerifierAuthParam)
	if err != nil {
		return nil, err
	}

	err = saveToken(*token, config)
	if err != nil {
		return nil, err
	}

	client := oauthConfig.Client(ctx, token)
	return client, nil
}

// saveToken copies the token into the configuration and saves it to cache
func saveToken(token oauth2.Token, config *Configuration) error {
	debug(*config, "Old token: "+config.Token.AccessToken)
	debug(*config, "New token: "+token.AccessToken)
	config.Token = token
	err := saveConfiguration(*config)
	if err != nil {
		return err
	}
	return nil
}

// checkAndSaveToken() takes an oauth2 token and refreshes and saves it if its expired
func checkAndSaveToken(token *oauth2.Token, config *Configuration) error {
	ctx, oauthConfig := getOauth2Config()
	if token.Expiry.Before(time.Now()) {
		debug(*config, "Token expired, refreshing...")
		tokenSource := oauthConfig.TokenSource(ctx, token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return fmt.Errorf("couldn't refresh token: %v", err)
		}
		if newToken.AccessToken != token.AccessToken {
			debug(*config, "Token has changed, saving new token...")
			saveToken(*newToken, config)
			token = newToken
		}
	} else {
		debug(*config, "Token not yet expired: "+token.Expiry.String())
	}
	return nil
}

// validateToken restores a token from the configuration and refreshes it
func validateToken(config *Configuration) (*http.Client, error) {
	ctx, oauthConfig := getOauth2Config()

	debug(*config, "Validating token...")

	err := checkAndSaveToken(&config.Token, config)
	if err != nil {
		return nil, err
	}

	client := oauthConfig.Client(ctx, &config.Token)
	return client, nil
}
