package main

//5699 links in 9 hrs...
//need to add links and possibly database

import (
	"fmt"
	"os"
	"time"
	"math/rand"
	"sync"
	"github.com/PuerkitoBio/goquery"
	"log"
	"strings"
	"errors"
	"reflect"
	"github.com/temoto/robotstxt"
	"net/http"
	"strconv"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"

)

type error interface {
    Error() string
}

var debug bool = true

type bgDoc struct {
	baseURL string
	url string
	doc *goquery.Document
}

type bg struct {
	name string
	price string
	url string
	InStock string
}

var bgs []bg

var crawledLinks []string 

var crawledSites int

var m *sync.Mutex = new(sync.Mutex)

type baseSite struct {
	url string
	r *robotstxt.Group
	crawlDelay time.Duration
	lastCrawl time.Time
	mDelay *sync.Mutex
}

var baseSites = make(map[string]*baseSite)


var baseURLs = []string{
		"http://www.adventgames.com.au",
		"http://www.gamesparadise.com.au",
		"http://www.milsims.com.au",}

func main() {
	//initialise logs
    f, err := os.OpenFile("bgLog", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
    if err != nil {
            log.Fatal(err)
    }   
    //defer to close when you're done with it, not because you think it's idiomatic!
    defer f.Close()
    //set output of logs to f
    log.SetOutput(f)

	//initialise base sites from robots.txt data
	var wg sync.WaitGroup

	for _ , url := range baseURLs{
		wg.Add(1)
		addBase(url,&wg)
	}
	wg.Wait()
	

	//initialise a channel with a suitable buffer
	buffer := 20000 * len(baseURLs)
	var cLinks chan string = make(chan string,buffer)
	defer close(cLinks)

	crawledSites = 0

	var cDocs chan bgDoc = make(chan bgDoc,10)
	defer close(cDocs)

	//go read_docs(cDocs) //get rid of function when I actually do something here

	//not sure how many crawlers to use here

	max_threads := 3
	for i:=0;i < max_threads;i++{
		wg.Add(1)
		go crawl(cLinks,cDocs, baseURLs, "crawler_"+strconv.Itoa(i),&wg)
	}


	for i:=0;i<3;i++{
		go crawlContent(cLinks, cDocs ,"content_crawler_"+strconv.Itoa(i))
	}


	for _, value := range baseURLs {
		cLinks <- value
	}

	go prioritiseCrawl(cLinks,cLinks,1000)

	// for i:=0;i<30;i++{
	// 	j:=rand.Intn(3)
	// 	fmt.Println(baseURLs[j])
	// 	cLinks <- baseURLs[j]
	// }

	wg.Wait()

	var input string
	fmt.Scanln(&input)

	//initialise crawlers with n= maxthreads crawlers
	
	// max_threads := 20
	// for i:=0;i < max_threads;i++{
	// 	wg.Add(1)
	// 	go crawl(c, baseURLs, max_threads,i,&wg)
	// }

	// //need a wait group here

	// //c <- "http://www.gamesparadise.com.au/the-settlers-of-catan-cities-knights-expansion"
	// //c <-"http://www.milsims.com.au/node/138106"
	// //c <- "http://www.adventgames.com.au/p/9230772/gloomhaven-preorder---2nd-printng---eta-18th-jan.html"
	// //c <- "http://www.milsims.com.au/node/138087"
	
	// for _, value := range baseURLs {
	// 	c <- value
	// }
	// c <- "http://www.milsims.com.au/catalog/1747"
	// go prioritiseCrawl(c,c,1000)
	// //don't want program to exit
	// // var input string
	// // fmt.Scanln(&input)
	// //implement waitgroup

	// wg.Wait() //wait for crawlers to finish
	// //amt:= time.Duration(3600*1000*3)
	// //time.Sleep(time.Millisecond *amt)
	// writeToFile()

	// fmt.Println("time Finished",time.Now())
}


//function 
func crawlContent(cLinks chan string, cDocs chan bgDoc,crawlerName string){

	defer log.Println("exiting Crawler :",crawlerName)

	for url := range cLinks {

		//get base YRL
		base, found := findBase(url)
		
		if found {
			//only crawl this url if allowed

			crawlAllowed(base)
			
	        doc, err := goquery.NewDocument(url)

	        //if a document is returned add it to the channel
	        if err != nil {

	        }

        	NewbgDoc := bgDoc{base,url,doc}
        	log.Println(crawlerName,url, time.Now().Format("15:04:05"))
			fmt.Println(crawlerName,url, time.Now().Format("15:04:05"))
        	//insert into db
        	insertlink(url)
        	
        	select {
			case cDocs <- NewbgDoc:
			case <-time.After(time.Second * 2):
		        log.Println("timeout after 2 secs")
				//not sure what to do here - log error?
			}
		}
    }
}

//function reads the robots.txt file from the site and adds all the neccesary information
//for us to use
func addBase(site string, wg *sync.WaitGroup){
	
	var b baseSite
	
	//get robots.txt file
	b.url = site
	resp, err :=  http.Get(b.url + "/robots.txt")
	
	if err != nil {
	    fmt.Println("Error accessing robots.txt")
	}

	defer resp.Body.Close()

	//feed it to 
	robots, err := robotstxt.FromResponse(resp)
	
	//what to do with errors here...?
	if err != nil {
	    fmt.Println("Error parsing robots.txt:")
	}

	group := robots.FindGroup("*")
	b.r = group
	b.crawlDelay = b.r.CrawlDelay

	//if there is no delay found in the robots.txt then set the default to 3 seconds - be nice!
	if b.crawlDelay < time.Duration(3000) {
		b.crawlDelay = time.Duration(time.Second*3)
	}

	b.lastCrawl = time.Now()

	b.mDelay = new(sync.Mutex)

	fmt.Println("added url: ",b.url," with delay : ",b.crawlDelay," at: ",b.lastCrawl)
	
	baseSites[site] = &b

	//completed 
	wg.Done()
}


func prioritiseCrawl(cCrawl chan string, cLinks chan string,nLinks int) {
	//var priorityLinks []string 
	var priorityLinks = []string{"http://www.milsims.com.au/node/",
		"http://www.adventgames.com.au/p/",
		"http://www.gamesparadise.com.au/board-games/",}//"/catalog/"
	//want to make sure sites are ordered by
	
	for {

		//sleep for 5 seconds
		amt:= time.Duration(4000)
		time.Sleep(time.Millisecond *amt)

		//take links out of cLinks shuffle and prioritise them
		cBlocked := false

		tempLinksCrawl :=[]string{}
		tempLinksDelay := []string{}

		for i:=0; i < len(cCrawl)/3 - 10; i++{
			//timeout if waiting to long so this does not block our crawlers
			
			select {
		    case link := <- cLinks:
		        if in_array(link, priorityLinks) {
					tempLinksCrawl = append(tempLinksCrawl,link)
				} else {
					tempLinksDelay= append(tempLinksDelay,link)
				}
		    case <-time.After(time.Second * 1):
		        fmt.Println("priority link timeout 1", i)
				cBlocked = true
		    }
		    //If the channel is now blocked get out of this loop
		    if cBlocked {break}	
		}

		for i := range tempLinksCrawl {
		    j := rand.Intn(i + 1)
		    tempLinksCrawl[i], tempLinksCrawl[j] = tempLinksCrawl[j], tempLinksCrawl[i]
		}

		for i:= range tempLinksCrawl {
			cCrawl <- tempLinksCrawl[i]
		}

		for i := range tempLinksDelay {
			cCrawl <- tempLinksDelay[i]
		}		
	}
}

func crawlAllowed(base string) {
	baseSites[base].mDelay.Lock()
	defer baseSites[base].mDelay.Unlock()
	for {
		if t:=time.Now(); t.Sub(baseSites[base].lastCrawl) > baseSites[base].crawlDelay {
			//baseSites[base].mDelay.Lock()
			baseSites[base].lastCrawl = t
			//baseSites[base].mDelay.Unlock()
			break
		}

		amt:= time.Duration(100 + rand.Intn(100))
		time.Sleep(time.Millisecond *amt)
	}

}

func crawl(cLinks chan string, cDocs chan bgDoc, allowedSites []string, crawlerName string, wg *sync.WaitGroup){
	
	var newbgDoc bgDoc

	defer fmt.Println("exiting Crawler :",crawlerName)
	
	for {
		finish_crawl:=false
		select {
		    case newbgDoc = <- cDocs:
		    case <-time.After(time.Second * 2000):
		    	//time.Sleep(time.Millisecond *3000)
		    	fmt.Println("Crawler Blocked")
		    	finish_crawl = true
			}

		if finish_crawl {break}

		base:=newbgDoc.baseURL
		content:=newbgDoc.doc
		url:= newbgDoc.url
		//download content from url
		//content := getContent(url)
		//get all links to crawl
		urls := getUrls(content,base)
		//add all urls to channel
		addUrls(cLinks,&urls,&crawledLinks)
		// //get all bg data from this url
		getbgData(content,url)	
	}
	wg.Done()
}

// func crawl(c chan string, allowedSites []string, max_threads int, crawlNo int, wg *sync.WaitGroup){
// 	//don't crawl links twice
// 	//can you check if a link exists - don't think you can just write to an array
// 	defer fmt.Println("exiting Crawler :",crawlNo)

// 	//amt:= time.Duration(2000)
// 	amt:= time.Duration(rand.Intn(3000))
// 	time.Sleep(time.Millisecond *amt)
// 	// for url := range c{

// 	// }
// 	//change for loop syntax to 
// 	for i := 0; i < 10000; i++ {

// 		t := time.Now()
		

// 		//get url to crawl from channel
// 		url:=""
// 		finish_crawl:=false
// 		select {
// 		    case url = <- c:
// 		    case <-time.After(time.Second * 2000):
// 		    	//time.Sleep(time.Millisecond *3000)
// 		    	fmt.Println("Crawler Blocked")
// 		    	finish_crawl = true
// 			}

// 		if finish_crawl {break}
// 		//find baseUrl
// 		base, found := findBase(url)

// 		//will return when a crawl is allowed -this should be paralised
// 		//change this to defer
		
// 		crawlAllowed(base)
		
// 		//fmt.Println("Crawl Allowed: ",base ,time.Now().Format("15:04:05"))

// 		//amt:= time.Duration(rand.Intn(max_threads*10000))
// 		//time.Sleep(time.Millisecond *amt)

// 		//if this is a base url we are interested in process/else discard
// 		if found {
// 			//for the moment print what is happening
			
// 			//download content from url
// 			content := getContent(url)
// 			//get all links to crawl
// 			urls := getUrls(content,url,base)
// 			//add all urls to channel
// 			addUrls(c,&urls,&crawledLinks)
// 			// //get all bg data from this url
// 			getbgData(content,url)	
			
// 			//put data into.. csv

// 			//add delay so as not to time out...x`
			
// 			//time.Sleep(time.Millisecond * 1000*max_threads)	
// 		}
		
// 		end := time.Now()
// 		crawledSites+= 1
// 		fmt.Println("crawler: ",crawlNo, end.Format("15:04:05"),end.Sub(t),crawledSites,i,len(crawledLinks),url)
		
// 	}
// 	wg.Done()
// }

func findBase(url string) (string, bool) {
	for _ , base := range baseURLs {
		if strings.Contains(url,base) {
			return base, true
		}
	}
	return "",false	
}


func getContent(url string) *goquery.Document {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func getUrls(doc *goquery.Document, base string) []string {
	
	var urls []string

	var interstr interface{}
	//need to add error checking
	//err := errors.New("Tag not found: "+ tag)

	doc.Find("a").Each(func(i int, s *goquery.Selection) {

	  	link, _ := s.Attr("href") //returns a string (must select what is inside)
	  
	  	interstr = link

	  	//check if a string is returned
	  	if _ , ok := interstr.(string);ok {
			if len(link) > 0 {
				if (string(link[0]) =="/"){
		  			link = base + link
			  	}

			  	//check if this link is allowed using robots.txt
			  	if  (baseSites[base].r.Test(strings.Replace(link, base, "", 1))) {
			  		if (strings.Contains(link,"http")) && !(strings.Contains(link,"javascript")) && (in_array(link,baseURLs)) {
				  		urls = append(urls,link)
			  		}
			  	}
			}		
		} 
  	})

	return urls
}

func addUrls(c chan string, urls *[]string, crawledLinks *[]string){	
	//for each of the urls found on this page
	for _, value := range *urls {
		//initialfalse

		if (strings.Contains(value,"http")) && !(strings.Contains(value,"javascript")) && (in_array(value,baseURLs)) {

			//check to see if this link has been found before
			found := false
			for _,link_value := range *crawledLinks {

				if value == link_value {
					found = true
				}
			}

			//if a new link has been found
			if found == false {
				//fmt.Println(value)
				//add this link to the channel for scraping
				if cap(c) - len(c) > 500 {
					select {
					case c <- value:
					default:
						//not sure what to do here
					}
					
					//lock and append this new link to our array
					m.Lock()
					*crawledLinks = append(*crawledLinks,value)
					m.Unlock()
				}
				
			}	
		}
		
	}

}

func in_array(val string, array []string) (exists bool) {
    exists = false

    for _, v := range array {

        if strings.Contains(val, v) {
            exists = true
            return
        }   
    }

    return
}


func getbgData(doc *goquery.Document, url string) {
	//want images -might want to get imagages
	var boardGame bg
	//boardGame = bg{"mage knight","100"}
	boardGame.url = url
	boardGame.name,boardGame.price,boardGame.InStock = "","",""

	if strings.Contains(url, "adventgames"){
		if strings.Contains(url, "/p/") {

			if name, err := findHtmlTag("div.product-title",doc); err!= nil {
				fmt.Print("No Data for URL: " + url + " and Tag: " + err.Error() + "\n")	
			} else {
				boardGame.name = name
			}

			if price, err := findHtmlTag("div.our-price",doc); err!= nil {
				fmt.Print("No Data for URL: " + url + " and Tag: " + err.Error() + "\n")	
			} else {
				boardGame.price = price
			}

			if InStock, err := findHtmlTag("div.product-currentstock",doc); err!= nil {
				fmt.Print("No Data for URL: " + url + " and Tag: " + err.Error() + "\n")	
			} else {
				boardGame.InStock = InStock
			}
		}
	} else if strings.Contains(url, "gamesparadise"){

			if name, err := findHtmlTag("div.product-shop.grid12-6.no-right-gutter h1",doc); err!= nil {
				fmt.Print("No Data for URL: " + url + " and Tag: " + err.Error() + "\n")	
			} else {
				boardGame.name = name
			}

			if price, err := findHtmlTag("div.product-shop.grid12-6.no-right-gutter span.price",doc); err!= nil {
				fmt.Print("No Data for URL: " + url + " and Tag: " + err.Error() + "\n")	
			} else {
				boardGame.price = price
			}

			if InStock, err := findHtmlTag("div.product-shop.grid12-6.no-right-gutter p.availability.in-stock span",doc); err!= nil {
				fmt.Print("No Data for URL: " + url + " and Tag: " + err.Error() + "\n")	
			} else {
				boardGame.InStock = InStock
			}
				
			
	} else if strings.Contains(url, "milsims") && strings.Contains(url, "/node/"){

		if name, err := findHtmlTag("h2.art-postheader",doc); err!= nil {
			fmt.Print("No Data for URL: " + url + " and Tag: " + err.Error() + "\n")	
		} else {
			boardGame.name = name
		}

		if price, err := findHtmlTag("span.uc-price-product.uc-price-display.uc-price",doc); err!= nil {
			fmt.Print("No Data for URL: " + url + " and Tag: " + err.Error() + "\n")	
		} else {
			boardGame.price = price
		}

		//find in stock /need to traverse through all "b's" to find words
		inStock:="In Stock"
		doc.Find("div#nerd-stats b").Each(func(i int, s *goquery.Selection) {
			if (strings.ToLower(s.Text()) == "out of stock"){
				inStock = "Out of Stock"
			}
		})
		boardGame.InStock = inStock
	}

	if boardGame.name != "" && boardGame.price != "" &&boardGame.InStock!= "" {
		fmt.Println(boardGame.name , boardGame.price ,boardGame.InStock)
		//bgs = append(bgs,boardGame)
		insertbg(&boardGame)
	}
}

func insertlink(url string) {
	now:= time.Now().UTC()
	//open conection to database
	db, err := sql.Open("sqlite3", "./bg.db")
	checkErr(err)
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO bgURLS(url,hasbg,dte) values(?,?,?)")
    checkErr(err)
    //insert data
    _ , err1 := stmt.Exec(url,"No", now.Format("2006-01-02 15:04:05"))
    checkErr(err1)

}

func insertbg(boardGame *bg){
	now:= time.Now().UTC()

	//open conection to database
	db, err := sql.Open("sqlite3", "./bg.db")
	checkErr(err)
	defer db.Close()

	stmt1, err := db.Prepare("DELETE FROM bg WHERE url = ?")
    checkErr(err)

    _, err1 := stmt1.Exec(boardGame.url)
    checkErr(err1)

    stmt2, err2 := db.Prepare("INSERT INTO bg(name,url,price,instock,dte) values(?,?,?,?,?)")
    checkErr(err2)
    //insert data
    _ , err3 := stmt2.Exec(boardGame.name, boardGame.url,boardGame.price,boardGame.InStock, now.Format("2006-01-02 15:04:05"))
    checkErr(err3)

    stmt3, err3 := db.Prepare("DELETE FROM bgURLS WHERE url = ?")
    checkErr(err3)

    _, err4 := stmt3.Exec(boardGame.url)
    checkErr(err4)

    stmt4, err5 := db.Prepare("INSERT INTO bgURLS(url,hasbg,dte) values(?,?,?)")
    checkErr(err5)
    //insert data
    _ , err6 := stmt4.Exec(boardGame.url,"Yes", now.Format("2006-01-02 15:04:05"))
    checkErr(err6)
    // _, err := res.LastInsertId()
    // checkErr(err)
}

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

func findHtmlTag(tag string, doc *goquery.Document) (string,error) {

	var interstr interface{}
	var text string
	err := errors.New("Tag not found: "+ tag)

	doc.Find(tag).Each(func(i int, s *goquery.Selection) {
			//will return the last tage found
			interstr = s.Text()
			
			if i , ok := interstr.(string);ok {
				err = nil
				text = i
			} else {
				err = errors.New("Type found is not a string. Type is " + reflect.TypeOf(interstr).String())
			}
		})

	return text, err 
}

func  writeToFile() {
	file, err := os.Create("result.txt")

    if err != nil {
        log.Fatal("Cannot create file", err)
    }
    defer file.Close()
	
	for _,val := range bgs {

		fmt.Fprintf(file,val.url + "\t"+ val.name + "\t"+ val.price + "\t"+ val.InStock +"\n")

	}

}