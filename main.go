package main

import (
	"flag"
	"fmt"
	"girls/zhihu"
	"math"
	"net/url"
	"os"
	"sync"
)

var queryUrl *string
var saveImage *bool

func init() {
	queryUrl = flag.String("u", "", "User home page")
	saveImage = flag.Bool("d", false, "Save images")
}

func main() {
	flag.Parse()

	if queryUrl == nil {
		fmt.Print("The URL is empty \n")
		os.Exit(0)
	}

	u, err := url.Parse(*queryUrl)
	if err != nil {
		panic(err)
	}

	client, err := zhihu.NewZhihuClient(*u)
	if err != nil{
		panic(err)
	}

	if err := client.CreateDirectory(); err != nil {
		panic(err)
	}

	imagesArr := make([]url.URL, 0)
	var page uint64 = 1

	for {
		data, err := client.Query(page)
		if err != nil {
			fmt.Printf("Page acquisition failed: %v\n", err)
			os.Exit(0)
		}

		count := client.GetAllAnswerCount(*data)
		if count == nil {
			fmt.Printf("Failed to get count: %v\n", err)
			os.Exit(0)
		}

		images, err := client.GetImages(*data)
		if err != nil {
			fmt.Printf("Failed to get image: %v\n", err)
			os.Exit(0)
		} else {
			for _, item := range images {
				imagesArr = append(imagesArr, item)
			}
		}

		allPageSize := uint64(math.Ceil(float64(*count) / 20))
		if page >= allPageSize {
			fmt.Printf("%v pages in total, searched: %v pages, found: %v photos\n", allPageSize, page, len(imagesArr))
			break
		} else {
			page += 1
		}
	}

	if saveImage != nil && *saveImage == true{
		miao := make(chan url.URL, 0)
		wg := sync.WaitGroup{}

		for i := 0; i < 5; i++ {
			go func(ch chan url.URL) {
				for s := range ch {
					if err := client.Download(s); err != nil {
						fmt.Printf("%v\n", err)
					}
					wg.Done()
				}
			}(miao)
		}

		for _, item := range imagesArr {
			miao <- item
			wg.Add(1)
		}

		wg.Wait()
	}else {
		if err := client.OutputTextFile(imagesArr); err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}
