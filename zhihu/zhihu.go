package zhihu

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

type Zhihu struct {
	urlToken string
	output   string
}

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

func NewZhihuClient(u url.URL) (*Zhihu, error) {
	res, err := http.Get(u.String())
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
	}

	for key := range json_data.InitialState.Entities.Users {
		token := json_data.InitialState.Entities.Users[key].UrlToken
		if token != nil {
			return &Zhihu{
				urlToken: *token,
				output: "./output",
			}, nil
		}
	}

	return nil, errors.New("UrlToken not found\n")
}

func (z Zhihu) Query(page uint64) (*OriginData, error) {
	httpUrl := url.URL{
		Scheme: "https",
		Host: "www.zhihu.com",
		Path: fmt.Sprintf("/people/%v/answers", z.urlToken),
	}
	queryValue := httpUrl.Query()
	queryValue.Set("page", strconv.FormatUint(page, 10))
	httpUrl.RawQuery = queryValue.Encode()

	res, err := http.Get(httpUrl.String())
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

func (z Zhihu) GetImages(data OriginData) ([]url.URL, error) {
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

	return z.removeDuplicate(imageArr), nil
}

func (z Zhihu) GetAllAnswerCount(data OriginData) *uint64 {
	var count *uint64
	for key := range data.InitialState.Entities.Users {
		count = data.InitialState.Entities.Users[key].AnswerCount
	}

	return count
}

func (z Zhihu) Download(u url.URL) error {
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

	fullPath, err := z.getFullDirectory()
	if err != nil{
		return err
	}
	return ioutil.WriteFile(filepath.Join(*fullPath, u.Path), buf, os.ModeAppend)
}

func (z Zhihu) getFullDirectory() (*string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(pwd, "output", z.urlToken)
	return &path, nil
}

func (z Zhihu) CreateDirectory() error {
	fullPath, err := z.getFullDirectory()
	if err != nil{
		return err
	}

	if _, err := os.Stat(*fullPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return nil
	}

	return os.MkdirAll(*fullPath, os.ModePerm)
}

func (z Zhihu) removeDuplicate(input []url.URL) []url.URL {
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

func (z Zhihu) OutputTextFile(images []url.URL) error {
	str := ""

	for _, item := range images{
		str += fmt.Sprintf("%v\n", item.String())
	}

	fullPath, err := z.getFullDirectory()
	if err != nil{
		return err
	}
	return ioutil.WriteFile(filepath.Join(*fullPath, "list.txt"), []byte(str), os.ModeAppend)
}
