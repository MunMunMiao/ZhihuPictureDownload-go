package main

import (
	"flag"
	"fmt"
	"github.com/gosuri/uiprogress"
	"math"
	"net/url"
	"os"
	"runtime"
	"sync"
	"zhihu/client"
)

var queryUrl *string
var saveImage *bool

func init() {
	queryUrl = flag.String("u", "", "User home page")
	saveImage = flag.Bool("d", false, "Save images")
}

func main() {
	flag.Parse()

	if queryUrl == nil || *queryUrl == "" {
		fmt.Print("The URL is empty \n")
		os.Exit(0)
	}

	u, err := url.Parse(*queryUrl)
	if err != nil {
		panic(err)
	}

	client, err := client.NewClient(*u)
	if err != nil{
		fmt.Printf("%v\n", err)
		os.Exit(0)
	}

	if err := client.CreateDirectory(); err != nil {
		panic(err)
	}

	imagesArr := make([]url.URL, 0)
	var page uint64 = 1

	fmt.Print("Parsing...\n")
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
			break
		} else {
			page += 1
		}
	}

	imagesArr = client.RemoveDuplicate(imagesArr)
	fmt.Printf("searched: %v pages, found: %v photos\n", page, len(imagesArr))

	if saveImage != nil && *saveImage == true{
		runtime.GOMAXPROCS(runtime.NumCPU())

		bar := uiprogress.AddBar(len(imagesArr)).AppendCompleted().PrependElapsed()
		bar.PrependFunc(func(b *uiprogress.Bar) string {
			return fmt.Sprintf("Saving (%d/%d)", b.Current(), len(imagesArr))
		})
		uiprogress.Start()

		miao := make(chan url.URL, 0)
		wg := sync.WaitGroup{}

		for i := 0; i < runtime.NumCPU(); i++ {
			go func(ch chan url.URL) {
				for s := range ch {
					if err := client.Download(s); err != nil {
						fmt.Printf("%v\n", err)
					}
					wg.Done()
					bar.Incr()
				}
			}(miao)
		}

		for _, item := range imagesArr {
			miao <- item
			wg.Add(1)
		}

		wg.Wait()
		uiprogress.Stop()
	}else {
		if err := client.OutputTextFile(imagesArr); err != nil {
			fmt.Printf("%v\n", err)
		}
	}

}
