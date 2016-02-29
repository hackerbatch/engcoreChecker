package main

import (
	//"crypto/rsa"
	"errors"
	//"flag"
	//"io/ioutil"
	//"log"
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
	"bytes"
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
	err := bow.Open("https://www.ubcengcore.com/students")
	if err != nil {
		return bow, err
	}
	// Click the login button
	err = bow.Click(".customContentContainer > p:nth-child(2) > a:nth-child(1)")
	if err != nil {
		return bow, err
	}
	
	// Log into shibboleth	
	// Should output "The University of British Columbia"
	fmt.Println(bow.Title())
	
	form, form_err := bow.Form("form[name='loginForm']")
	if form_err != nil {
		return bow, form_err
	}

	form.Input("j_username", u.Username)
	form.Input("j_password", u.Password)
	
	err = form.Submit()
	if err != nil {
		return bow, err
	}

	jar, err := cookiejar.New(nil)
	jar.SetCookies(bow.Url(), bow.SiteCookies())

	if err != nil {
		return bow, err
	}
	client := &http.Client{
		Jar: jar,
	}
	
	// Continue SAML
	r := bytes.NewReader([]byte( bow.Body() ))
	doc, err := goquery.NewDocumentFromReader(r)
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
		fmt.Println("Error with response3")
		return bow, err
	}
	fmt.Println("got past SAML")
	defer resp3.Body.Close()
	
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
	
	_, err := user.loginToEngCore()
	fmt.Println("Finished logging into EngCore")
	if err != nil {
		fmt.Println("Error is: " +err.Error())
		panic(err)
	}

	//pingEngCore(bow)
	// output logs to the terminal
	//for i := range c {
	//	fmt.Println(i)
	//}

	fmt.Println("Done")
	return

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
