package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Client struct {
	urlToken string
	output   string
}

const userAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"

type OriginData struct {
	InitialState struct {
		Entities struct {
			Answers map[uint64]struct {
				Content string `json:"content"`
			} `json:"answers"`
			Users map[string]struct {
				AnswerCount *uint64 `json:"answerCount,omitempty"`
				UrlToken    *string `json:"urlToken,omitempty"`
			} `json:"users"`
		} `json:"entities"`
	} `json:"initialState"`
}

func NewClient(u url.URL) (*Client, error) {
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("user-agent", userAgent)

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("status code error: %d\n%s\n", res.StatusCode, res.Status))
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	json_str := doc.Find("#js-initialData").Text()
	json_data := OriginData{}
	if err := json.Unmarshal([]byte(json_str), &json_data); err != nil {
		return nil, err
	}

	for key := range json_data.InitialState.Entities.Users {
		token := json_data.InitialState.Entities.Users[key].UrlToken
		if token != nil {
			return &Client{
				urlToken: *token,
				output:   "./output",
			}, nil
		}
	}

	return nil, errors.New("UrlToken not found\n")
}

func (c Client) Query(page uint64) (*OriginData, error) {
	httpUrl := url.URL{
		Scheme: "https",
		Host:   "www.zhihu.com",
		Path:   fmt.Sprintf("/people/%v/answers", c.urlToken),
	}
	queryValue := httpUrl.Query()
	queryValue.Set("page", strconv.FormatUint(page, 10))
	httpUrl.RawQuery = queryValue.Encode()

	req, _ := http.NewRequest("GET", httpUrl.String(), nil)
	req.Header.Set("user-agent", userAgent)

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("status code error: %d %s", res.StatusCode, res.Status))
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	json_str := doc.Find("#js-initialData").Text()
	json_data := OriginData{}
	if err := json.Unmarshal([]byte(json_str), &json_data); err != nil {
		return nil, err
	} else {
		return &json_data, nil
	}
}

func (c Client) GetImages(data OriginData) ([]url.URL, error) {
	imageArr := make([]url.URL, 0)

	for key := range data.InitialState.Entities.Answers {
		content := data.InitialState.Entities.Answers[key].Content
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err != nil {
			return imageArr, err
		}

		doc.Find("img").Each(func(i int, selection *goquery.Selection) {
			for i := range selection.Nodes {
				image_url, _ := selection.Eq(i).Attr("data-original")
				if len(image_url) > 0 {
					u, _ := url.Parse(image_url)
					u.RawQuery = ""

					reg := regexp.MustCompile("/_r|_hd|\\/80|\\/50|_720w|_14400w|_b|_xll|_xl+/g")
					u.Path = reg.ReplaceAllString(u.Path, "")
					imageArr = append(imageArr, *u)
				}
			}
		})
	}
	return imageArr, nil
}

func (c Client) GetAllAnswerCount(data OriginData) *uint64 {
	var count *uint64
	for key := range data.InitialState.Entities.Users {
		count = data.InitialState.Entities.Users[key].AnswerCount
	}

	return count
}

func (c Client) Download(u url.URL) error {
	res, err := http.Get(u.String())
	if err != nil {
		return err
	}

	defer res.Body.Close()

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	size, err := strconv.ParseInt(res.Header.Get("content-length"), 10, 64)
	if err != nil {
		return err
	}

	if int64(len(buf)) != size {
		return errors.New("Incomplete data\n")
	}

	fullPath, err := c.getFullDirectory()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(*fullPath, u.Path), buf, os.ModeAppend)
}

func (c Client) getFullDirectory() (*string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(pwd, "output", c.urlToken)
	return &path, nil
}

func (c Client) CreateDirectory() error {
	fullPath, err := c.getFullDirectory()
	if err != nil {
		return err
	}

	if _, err := os.Stat(*fullPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return nil
	}

	return os.MkdirAll(*fullPath, os.ModePerm)
}

func (c Client) RemoveDuplicate(input []url.URL) []url.URL {
	result := make([]url.URL, 0)
	temp := map[string]url.URL{}

	for _, item := range input {
		if _, ok := temp[item.Path]; !ok {
			temp[item.Path] = item
			result = append(result, item)
		}
	}

	return result
}

func (c Client) OutputTextFile(images []url.URL) error {
	str := ""

	for _, item := range images {
		str += fmt.Sprintf("%v\n", item.String())
	}

	fullPath, err := c.getFullDirectory()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(*fullPath, "list.txt"), []byte(str), os.ModePerm)
}
