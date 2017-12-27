package main

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
)


type error interface {
    Error() string
}

var debug bool = true

type bg struct {
	name string
	price string
	url string
	InStock string
}

var bgs []bg

var crawledLinks []string 

var m *sync.Mutex = new(sync.Mutex)

var baseURLs = []string{
		"http://www.adventgames.com.au",
		"http://www.gamesparadise.com.au",
		"http://www.milsims.com.au"}


func main() {
	//robots.txt, wait time so I don't denial of service //wait time

	//initialise a channel with a bugger
	buffer := 10000
	var c chan string = make(chan string,buffer)
	defer close(c)


	//initialise crawlers with n= maxthreads crawlers
	max_threads := 10
	for i:=0;i < max_threads;i++{
		go crawl(c, baseURLs, max_threads)	
	}
	c <- "http://www.gamesparadise.com.au/the-settlers-of-catan-cities-knights-expansion"
	c <-"http://www.milsims.com.au/node/138106"
	c <- "http://www.adventgames.com.au/p/9230772/gloomhaven-preorder---2nd-printng---eta-18th-jan.html"
	//c <- "http://www.milsims.com.au/node/138087"
	
	// for _, value := range baseURLs {
	// 	c <- value
	// }
	
	//don't want program to exit
	var input string
	fmt.Scanln(&input)
	writeToFile()
}


func crawl(c chan string, allowedSites []string, max_threads int){
	//don't crawl links twice
	//can you check if a link exists - don't think you can just write to an array
	
	for i := 0; i < 100000; i++ {
		//get url to crawl from channel
		url := <- c

		//find baseUrl
		base, found := findBase(url)
		//if this is a base url we are interested in process/else discard
		if found {
			//for the moment print what is happening
			

			fmt.Println(i,len(crawledLinks),url)
			

			//download content from url
			content := getContent(url)
			//get all links to crawl
			urls := getUrls(content,url,base)
			//add all urls to channel
			addUrls(c,&urls,&crawledLinks)
			// //get all bg data from this url
			getbgData(content,url)	
			
			//put data into.. csv

			//add delay so as not to time out...
			
			//time.Sleep(time.Millisecond * 1000*max_threads)
			amt:= time.Duration(rand.Intn(7000))
			time.Sleep(time.Millisecond *amt)
		}
		
	}

}

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

func getUrls(doc *goquery.Document, url string, base string) []string {
	
	var urls []string

	doc.Find("a").Each(func(i int, s *goquery.Selection) {


	  	link, _ := s.Attr("href") //returns a string (must select what is inside)
	  	if (strings.Contains(link,"http")) && !(strings.Contains(link,"javascript")) && (in_array(link,baseURLs)) {
		  	if (string(link[0]) =="/"){
		  		//find the base 
		  		urls = append(urls,base+link)
		  	} else {
		  		urls = append(urls,link)
		  	}
	  	}
	  	

  	})

	return urls
}

func addUrls(c chan string, urls *[]string, crawledLinks *[]string){	
	//for each of the urls found on this page
	for _, value := range *urls {
		//initialise found to false
		found := false

		if (strings.Contains(value,"http")) && !(strings.Contains(value,"javascript")) && (in_array(value,baseURLs)) {

			//if the link found on this page has already been found before set the found flag
			//link should only contain 1000s..? Exit if True?
			for _,link_value := range *crawledLinks {

				if value == link_value {
					found = true
				}
			}

			//if a new link has been found
			if found == false {
				//fmt.Println(value)
				//add this link to the channel for scraping
				c <- value
				//lock and append this new link to our array
				m.Lock()
				*crawledLinks = append(*crawledLinks,value)
				m.Unlock()
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
		bgs = append(bgs,boardGame)
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