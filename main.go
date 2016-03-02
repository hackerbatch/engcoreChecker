package main

import (
	//"crypto/rsa"	
	"encoding/json"
	"errors"
	"flag"
	"strings"
	//"io/ioutil"
	"log"
	//"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
	//"strconv"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/jar"
	"github.com/PuerkitoBio/goquery"
	"github.com/jinzhu/gorm"
	"github.com/headzoo/surf/browser"
	_ "github.com/mattn/go-sqlite3"
	"fmt"
	//"bytes"
)

type User struct {
	Username, Password             string
	LastChecked		       time.Time
	Encrypted		       bool
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

func (u *User) loginToEngCore() (*browser.Browser, error) {
	bow := surf.NewBrowser()
	bow.SetCookieJar(jar.NewMemoryCookies())
	bow.SetAttributes(browser.AttributeMap{
		browser.SendReferer:         true,
		browser.MetaRefreshHandling: true,
		browser.FollowRedirects:     true,
	})
	
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
	//body, _ := ioutil.ReadAll(resp.Body)

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
	//body, _ = ioutil.ReadAll(resp3.Body)
	
	bow.SetCookieJar(client.Jar)
	resp5, err := client.Get("https://www.ubcengcore.com/secure/shibboleth.htm")
	if err != nil {
		return bow, err
	}
	defer resp5.Body.Close()
	//body, _ = ioutil.ReadAll(resp5.Body)
	//fmt.Println(string(body))
	
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

type PingRecord struct {
	lagTime		string
	url		string
}

var (
         c = make(chan *PingRecord)
)


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
	
	//timeoutStart := time.Now()
	var totalPing int64 = 0
	var numPing int64 = 0
	//var avgPing int64 = 0

	//Loop while our seesion hasn't expired
	for err == nil {
		//lag timer start
		start := time.Now()

		res, err := client.Get(pingUrl)
		if err != nil {
			msg := "Error:" + err.Error()
			//c <- msg
			log.Println(msg)
			fmt.Println(msg)
			break	
		} else {
			lag := time.Since(start)
			//var msg string
			currPing := &PingRecord{
				url: pingUrl,
			}		
			
			numPing++
			totalPing = totalPing + int64(lag) 

			//	running slow
			if lag > time.Duration(300)*time.Second {
				//msg = pingUrl + " lag: " + lag.String()
				//currPing.lagTime = lag.String()	
			}	

			//msg = pingUrl + ", lag: " + lag.String()
			//c <- msg
	
			currPing.lagTime = lag.String()
			c <- currPing	
	
			res.Body.Close()
		}
	}
	
	//timeoutPeriod := time.Since(timeoutStart)
	//avgPing = totalPing/numPing
	
	//This is a bit hacky
	//msg := "Timeout Period was:" + timeoutPeriod.String() + " average ping lag: " + strconv.FormatInt(avgPing, 10) 
	//c <- msg
	close(c) 
}

var addr = flag.String("addr", ":3000", "The address to listen on.")

func main() {	
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
	
	//go pollActivator(db, key)
	//go activateEverything(db, key)

	
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/api/v1/getPoint", func(w http.ResponseWriter, r *http.Request) {
		val := <-c
		if val == nil {
			http.Error(w, "No data could be sent", 400)
			return
		} else {
			js, err := json.Marshal(val)
			
			if err != nil {
    				http.Error(w, err.Error(), http.StatusInternalServerError)
    				return
  			}			
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		}
	
	})

	http.HandleFunc("/api/v1/check", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		user := &User{
			Username:   r.FormValue("username"),
			Password:   r.FormValue("password"),
		}
		/*		
		user := &User{
			Username: "user",
			Password: "pass",
		}
		*/
			
		if err := user.Validate(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		
		bow, err := user.loginToEngCore()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				log.Println("UBC EngCore login throttling error")
				http.Error(w, "UBC EngCore login throttling", 400)
				return
			} else {
				log.Println("Error is: " +err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
		}

		go pingEngCore(bow)
		//for i := range c {
		//	fmt.Println(i)
		//}
		
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
}
