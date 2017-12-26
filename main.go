package main

import (
	"fmt"
	//"time"
	//"math/rand"
	"sync"
	"github.com/PuerkitoBio/goquery"
	"log"
	"strings"
//	"reflect"
)

//minor change

// type bg struct {
// 	name string
// 	price string
// 	inStock string
// 	supplier string
// 	url string
// }

// var bgs []bg

// type bgSite struct {
// 	url string
// 	siteMap bool
// }

var crawledLinks []string 
var test string
var m *sync.Mutex = new(sync.Mutex)

var baseURLs = []bgSite{
		"http://www.adventgames.com.au",
		"http://www.gamesparadise.com.au",
		"http://www.milsims.com.au"}




func main() {
	//robots.txt, wait time so I don't denial of service //wait time

	buffer := 10000
	var c chan string = make(chan string,buffer)
	defer close(c)
	//m = new(sync.Mutex)

	//max threads used
	max_threads := 10

	//set first site to crawl
	// bgStie := "http://www.adventgames.com.au/"
	// bgStie = "http://www.adventgames.com.au/p/9287023/planet-of-the-apes.html"
	// bgStie = "http://www.gamesparadise.com.au/small-world-legends-tales-expansion"

	//crawlSiteMap
	//get sitemap data - if there is a sitemap no need to crawl

	//find links //how many concurrent links can I do?
	for i:=0;i < max_threads;i++{
		go crawl(c, baseURLs, max_threads)	
	}
	c <- "http://www.gamesparadise.com.au/board-games"
	//c <- "http://www.gamesparadise.com.au"
	
	// for _, value := range allowedSites {
	// 	c <- value
	// }
	
	//don't want program to exit
	var input string
	fmt.Scanln(&input)
}

// func shuffleChan(c chan string,){
	

// 	i:=0
// 	amt:= time.Duration(rand.Intn(500))
// 	time.Sleep(time.Millisecond *amt)
// 	for {
// 		msg:= <- c
// 		i+=1
// 		fmt.Println(i,msg)

// 	}
// }

func crawl(c chan string, allowedSites []string, max_threads int){
	//don't crawl links twice
	//can you check if a link exists - don't think you can just write to an array
	
	for i := 0; i < 100; i++ {
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
			fmt.Println(i,len(crawledLinks),url)
			
			time.Sleep(time.Millisecond * 1000*max_threads)

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
	//var boardGame bg
	if strings.Contains(url, "adventgames"){
		
		if strings.Contains(url, "/p/") {
			doc.Find("div.our-price").Each(func(i int, s *goquery.Selection) {
				test := s.Find("div").Text()
				fmt.Println(i,test)
			  })

			//find title
			doc.Find("div.product-title").Each(func(i int, s *goquery.Selection) {
				test := s.Text()
			  	fmt.Println(i,test)
			})

			//find in stock
			doc.Find("div.product-currentstock").Each(func(i int, s *goquery.Selection) {
				test := s.Text()
			  	fmt.Println(i,test)
			})
		}
	} else if strings.Contains(url, "gamesparadise"){
			//doc.Find("div.main.container.show-bg")
			doc.Find("div.product-shop.grid12-6.no-right-gutter").Each(func(i int, s *goquery.Selection) {
				
				// s.Find("div").Each(func(j int, sel *goquery.Selection) {
				// 	test := sel.Text()
				// 	fmt.Println(j,test)
				// })

				s.Find("span.price").Each(func(j int, sel *goquery.Selection) {
					test := sel.Text()
					fmt.Println(j,test)
				})

				s.Find("h1").Each(func(j int, sel *goquery.Selection) {
					test := sel.Text()
					fmt.Println(j,test)
				})

				s.Find("p.availability.in-stock span").Each(func(j int, sel *goquery.Selection) {
					test := sel.Text()
					fmt.Println(j,test)
				})

			  })
	} else if strings.Contains(url, "milsims") && strings.Contains(url, "/node/"){
		doc.Find("span.uc-price-product.uc-price-display.uc-price").Each(func(i int, s *goquery.Selection) {
			test := s.Text()
			fmt.Println(i,test)
		  })

		//find title
		doc.Find("h2.art-postheader").Each(func(i int, s *goquery.Selection) {
			test := s.Text()
		  	fmt.Println(i,test)
		})

		//find in stock /need to traverse through all "b's" to find words
		inStock:="In Stock"
		doc.Find("div#nerd-stats b").Each(func(i int, s *goquery.Selection) {
			if (strings.ToLower(s.Text()) == "out of stock"){
				inStock = "Out of Stock"
			}
		})
		fmt.Println(inStock)
	}
}