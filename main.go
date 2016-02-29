package main

import (
	//"crypto/rsa"
	"errors"
	//"flag"
	"strings"
	"io/ioutil"
	//"log"
	//"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
	"strconv"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/jar"
	"github.com/PuerkitoBio/goquery"
	//"github.com/jinzhu/gorm"
	"github.com/headzoo/surf/browser"
	//_ "github.com/mattn/go-sqlite3"
	"fmt"
	//"bytes"
)

type User struct {
	Username, Password             string
}

func (u User) Validate() error {
	if len(u.Username) == 0 {
		return errors.New("Invalid username.")
	}
	if len(u.Password) == 0 {
		return errors.New("Invalid password.")
	}
	return nil
}

var (
	c = make(chan string)
)

func (u *User) loginToEngCore() (*browser.Browser, error) {
	bow := surf.NewBrowser()
	bow.SetCookieJar(jar.NewMemoryCookies())
	bow.SetAttributes(browser.AttributeMap{
		browser.SendReferer:         true,
		browser.MetaRefreshHandling: true,
		browser.FollowRedirects:     true,
	})
	/*	
	err := bow.Open("https://www.ubcengcore.com/secure/shibboleth.htm") 
	if err != nil {
		return bow, err
	}
	// Click the login button
	bow.Click(".customContentContainer > p:nth-child(2) > a:nth-child(1)")
	if err != nil {
		return bow, err
	}
	*/
	
	/*	
	// Log into shibboleth	
	// Should output "The University of British Columbia"
	fmt.Println(bow.Title())
	fmt.Println(bow.Url().String())
	form, form_err := bow.Form("form[name='loginForm']")
	if form_err != nil {
		return bow, form_err
	}
	
	actionUrl, err := url.Parse(form.Action())
	host, _, _ := net.SplitHostPort(actionUrl.Host)
	fmt.Println("modified url: https://" + host + actionUrl.Path)
	
	form.Input("j_username", u.Username)
	form.Input("j_password", u.Password)
	
	err = form.Submit()
	fmt.Println("about to submit form")
	if err != nil {
		return bow, err
	}
	fmt.Println("submitted form")
	fmt.Println(bow.Url().String())
	
	form, form_err = bow.Form("form")
	if form_err != nil {
		return bow, form_err
	}
	
	relayState, _ := bow.Dom().Find("input[name='RelayState']").Attr("value")
	//fmt.Println("RelayState Value: " + relayState)
	form.Input("RelayState", relayState)
	
	samlResponse, _ := bow.Dom().Find("input[name='SAMLResponse']").Attr("value")
	form.Input("SAMLResponse", samlResponse)
	form.Set("action", "Continue")
	fmt.Println("Form action: " + form.Action())
	
	err = form.Submit()
	if err != nil {
		return bow, err
	}
	*/
	
	jar, err := cookiejar.New(nil)
	if err != nil {
		return bow, err
	}
	client := &http.Client{
		Jar: jar,
		Timeout: time.Duration(5 * time.Second),
	}
	
	resp, err := client.Get("https://ubcengcore.com/secure/shibboleth.htm")
	if err != nil {
		return bow, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	// Log into shibboleth
	resp2, err := client.PostForm("https://shibboleth2.id.ubc.ca/idp/Authn/UserPassword", url.Values{
		"j_username": {u.Username},
		"j_password": {u.Password},
		"action":     {"Continue"},
	})
	if err != nil {
		return bow, err
	}
	defer resp2.Body.Close()

	// Continue SAML
	doc, err := goquery.NewDocumentFromResponse(resp2)
	if err != nil {
		return bow, err
	}
	action := doc.Find("form").AttrOr("action", "")
	relayState := doc.Find("input[name=\"RelayState\"]").AttrOr("value", "")
	samlResponse := doc.Find("input[name=\"SAMLResponse\"]").AttrOr("value", "")

	resp3, err := client.PostForm(action, url.Values{
		"RelayState":   {relayState},
		"SAMLResponse": {samlResponse},
		"action":       {"Continue"},
	})
	if err != nil {
		return bow, err
	}
	defer resp3.Body.Close()
	body, _ = ioutil.ReadAll(resp3.Body)
	fmt.Println(string(body))	
	bow.SetCookieJar(client.Jar)
	resp5, err := client.Get("https://www.ubcengcore.com/secure/shibboleth.htm")
	if err != nil {
		return bow, err
	}
	defer resp5.Body.Close()
	body, _ = ioutil.ReadAll(resp5.Body)
	fmt.Println(string(body))
	
	resp6, err := client.Get("https://www.ubcengcore.com/myAccount")
	if err != nil {
		return bow, err
	}
	defer resp6.Body.Close()
	
	err = bow.Open("https://www.ubcengcore.com/myAccount")
        if err != nil {
                return bow, err
        }
	return bow, nil
}

//Record engcore pingback times
func pingEngCore(bow *browser.Browser) {

	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	jar.SetCookies(bow.Url(), bow.SiteCookies())


	//Test that we can ping site without redirects
	client := &http.Client{
		Jar: jar,
	}

	//Url that we should use to test pingback 
	pingUrl := bow.Url().String()
	if err != nil {
		panic(err)	
	}
	
	timeoutStart := time.Now()
	var totalPing int64 = 0
	var numPing int64 = 0
	var avgPing int64 = 0

	//Loop while our seesion hasn't expired
	for err == nil {
		//lag timer start
		start := time.Now()

		res, err := client.Get(pingUrl)
		if err != nil {
			msg := "Error:" + err.Error()
			c <- msg
		} else {
			lag := time.Since(start)
			var msg string
			
			numPing++
			totalPing = totalPing + int64(lag) 

			//	running slow
			if lag > time.Duration(300)*time.Second {
				msg = pingUrl + " lag: " + lag.String()
			}

			msg = pingUrl + ", lag: " + lag.String()
			c <- msg

			res.Body.Close()
		}
	}
	
	timeoutPeriod := time.Since(timeoutStart)
	avgPing = totalPing/numPing
	
	//This is a bit hacky
	msg := "Timeout Period was:" + timeoutPeriod.String() + " average ping lag: " + strconv.FormatInt(avgPing, 10) 
	c <- msg
	close(c) 
}
/*
	func activateEverything(db gorm.DB, key *rsa.PrivateKey) {
		log.Println("Checking for new UPasses...")
		var users []*User
		db.Find(&users)
		for i, user := range users {
			if err := user.Decrypt(key); err != nil {
				log.Printf("ERR decrypting %s, %s", user.Username, err)
				continue
			}
			if err := user.Activate(); err != nil {
				log.Printf("ERR activating %s, %s", user.Username, err)
				continue
			}
			if err := user.Decrypt(key); err != nil {
				log.Printf("ERR decrypting %s, %s", user.Username, err)
				continue
			}
			db.Model(user).Update("last_activated", user.LastActivated)
			// Remove decrypted version from memory.
			users[i] = nil
		}
	}

	func pollActivator(db gorm.DB, key *rsa.PrivateKey) {
		ticker := time.NewTicker(24 * time.Hour)
		for _ = range ticker.C {
			activateEverything(db, key)
		}
	}

	var addr = flag.String("addr", ":3000", "The address to listen on.")
*/
func main() {
	user := &User{
		Username: "davidb7",
		Password: "ptW7$7MM",
	}
	
	bow, err := user.loginToEngCore()
	fmt.Println("Finished logging into EngCore")
	if err != nil {
		if strings.Contains(err.Error(), "use of closed network connection") {
			fmt.Println("UBC EngCore throttling error")
			return
		} else {
			fmt.Println("Error is: " +err.Error())
			panic(err)
		}
	}

	go pingEngCore(bow)
	// output logs to the terminal
	for i := range c {
		fmt.Println(i)
	}

	fmt.Println("Done")

	/*
	flag.Parse()

	db, err := gorm.Open("sqlite3", "./user.db")
	if err != nil {
		log.Fatal(err)
	}
	db.CreateTable(&User{})
	db.AutoMigrate(&User{})

	key, err := readKeyOrGenerate("./db.key")
	if err != nil {
		log.Fatal(err)
	}
	
	go pollActivator(db, key)
	go activateEverything(db, key)

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/api/v1/test", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		user := &User{
			Username:   r.FormValue("username"),
			Password:   r.FormValue("password"),
		}
		if err := user.Validate(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := user.Activate(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := user.Encrypt(key); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err := db.Create(user).Error; err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("Successfully created renewer."))
	})
	log.Printf("Listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
	*/
}
