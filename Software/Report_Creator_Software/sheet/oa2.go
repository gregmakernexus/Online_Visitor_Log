package sheet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"example.com/debug"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Oauth2Client struct {
	Oauth2Config *oauth2.Config
	Tok          *oauth2.Token
}

func NewOauth2Client(ctx context.Context, l *debug.DebugClient) (*Oauth2Client, error) {
	var err error
	o := &Oauth2Client{}
	log = l
	// The client secret is a downloaded file from the GCP Console.
	// Instead of reading it from a file it is initialized here.
	secret := `{"installed":{"client_id":"624037715561-nfua183o9tv013fjto1vame4csgf7tjt.apps.googleusercontent.com",
				"project_id":"makernexus",
				"auth_uri":"https://accounts.google.com/o/oauth2/auth",
				"token_uri":"https://oauth2.googleapis.com/token",
				"auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs",
				"client_secret":"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",
				"redirect_uris":["http://localhost:8090"]}}`
	// Build the oauth2 config that is used to create the client
	scopes := []string{"https://www.googleapis.com/auth/spreadsheets", "https://www.googleapis.com/auth/drive"}
	o.Oauth2Config, err = google.ConfigFromJSON([]byte(secret), scopes...)
	if err != nil {
		log.V(0).Fatalf("Unable to parse client secret file to config: %v", err)
	}
	return o, err
}

// Google will send an authentication request to this url.  It automatically responds positive
func (o *Oauth2Client) authServer(ctx context.Context) {
	/*-----------------------------------------------------
	 *  HTTP server call back to catch the authentication requests from google
	 *  and exchange the auth code to prove who we are, and get oauth2 tokens
	 *-----------------------------------------------------*/
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var err error
		if ctx.Value("logLevel") == 1 {
			log.V(0).Println(req)
		}
		// Extract the authorization code from the request
		code := req.URL.Query().Get("code")
		if code == "" {
			log.V(0).Printf("HTTP request had no authorization code.  Ignored.\n")
			return
		}
		// Exchange the code with google to get a oauth2 token
		o.Tok, err = o.Oauth2Config.Exchange(ctx, code)
		if err != nil {
			// Send error to Browser
			fmt.Fprintf(w, "oauth2.exchange error:\n%v\n", err)
			log.V(0).Printf("oauth2 error:\n%v\n", err)
			return
		}
		// Save oauth2 token to file and send success to browser
		o.saveToken("token.json")
		fmt.Fprintf(w, "User successfully authenticated\n")
		log.V(0).Println("")
		log.V(0).Printf("authentication Successful\n")
	})

	// This a HTTP server thread.  There is no return from here
	// will continue to run once started till the program ends
	log.V(1).Printf("Authentication Server Started\n")
	http.ListenAndServe(":8090", nil)
}

// Request a token from the web, then returns the retrieved token.
func (o *Oauth2Client) GetTokenFromWeb(ctx context.Context) {
	log.Println("Get User Permission for Google Services")

	// Build the url for to identify the credentials that will be used to access
	// the spreadsheet.  The credentials will need access to the "makernexus" project
	go o.authServer(ctx)
	authURL := o.Oauth2Config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	var cmd string
	var args []string
	// Start the browser with the url
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, authURL)
	exec.Command(cmd, args...).Start()
	// Wait for user to give permission via chrome.  The
	// oauth2 token is stored by the authorization server
	for i := 1; i < 600; i++ {
		if o.Tok != nil {
			return
		}
		time.Sleep(time.Second)
	}
	log.V(0).Fatal("Timeout waiting for user permission")

}

// Retrieves a token from a local file.
func (o *Oauth2Client) GetTokenFromFile(file string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("can't get home directory:%v", err)
	}
	if err := os.Chdir(filepath.Join(home, ".makerNexus")); err != nil {
		return fmt.Errorf("can't cd to directory:%v", err)
	}
	log.V(1).Printf("Reading Token:%v\n", file)
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("gettokenfromfile error:%v", err)
	}
	defer f.Close()
	o.Tok = &oauth2.Token{}
	if err = json.NewDecoder(f).Decode(o.Tok); err != nil {
		return fmt.Errorf("gettokenfromfile error:%v", err)
	}
	log.V(1).Printf("Read Token:%v\n", o.Tok)
	return err
}

// Saves a token to a file path.
func (o *Oauth2Client) saveToken(path string) {
	log.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.V(0).Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(o.Tok)
}
