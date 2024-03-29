package pokemonsleep

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	vision "cloud.google.com/go/vision/apiv1"
	"go.uber.org/zap"
)

type Client struct {
	SlackToken string `json:"-"`

	Vision *vision.ImageAnnotatorClient `json:"-"`
	Logger *zap.Logger                  `json:"-"`

	Foods  []*Food `json:"foods"`
	Salad  []*Cook `json:"salad"`
	Desert []*Cook `json:"desert"`
	Curry  []*Cook `json:"curry"`
}

func NewClientFromRemote(ctx context.Context, token string, foodsConfigUrl, cooksConfigUrl string, logger *zap.Logger) (*Client, error) {
	ret := &Client{
		SlackToken: token,
		Logger:     logger,
	}

	// vision clientの初期化
	vc, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("init vision client failed: %w", err)
	}
	ret.Vision = vc

	// json config読み込み
	err = LoadJsonConfig(foodsConfigUrl, ret)
	if err != nil {
		return nil, fmt.Errorf("load json config (%s) failed: %w", foodsConfigUrl, err)
	}
	err = LoadJsonConfig(cooksConfigUrl, ret)
	if err != nil {
		return nil, fmt.Errorf("load json config (%s) failed: %w", cooksConfigUrl, err)
	}

	ret.Logger.Info("init Client.")
	return ret, nil
}

func NewClientFromLocal(ctx context.Context, token string, foodsConfigPath, cooksConfigPath string, logger *zap.Logger) (*Client, error) {
	ret := &Client{
		SlackToken: token,
		Logger:     logger,
	}

	// vision clientの初期化
	vc, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("init vision client failed: %w", err)
	}
	ret.Vision = vc

	// json config読み込み
	err = LoadJsonConfig(foodsConfigPath, ret)
	if err != nil {
		return nil, fmt.Errorf("load json config (%s) failed: %w", foodsConfigPath, err)
	}
	err = LoadJsonConfig(cooksConfigPath, ret)
	if err != nil {
		return nil, fmt.Errorf("load json config (%s) failed: %w", cooksConfigPath, err)
	}

	ret.Logger.Info("init Client.")
	return ret, nil
}

func (c *Client) Close() {
	c.Vision.Close()
}

func (c *Client) GetResultText(ctx context.Context, text, filetype, imageUrl string, originalW, originalH int) ([]string, error) {
	resp, err := DownloadImage(imageUrl, c.SlackToken)
	if err != nil {
		return nil, fmt.Errorf("download image failed: %w", err)
	}
	defer resp.Body.Close()

	img, err := NewImage(resp.Body, filetype, originalW, originalH, c.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Image:%w", err)
	}

	dres, err := c.OCR(ctx, img)
	if err != nil {
		return nil, fmt.Errorf("failed OCR:%w", err)
	}

	dres.DetectFoods(c.Foods)

	var ret []string
	var foodsStr string
	for k, v := range dres.DetectedFoods {
		foodsStr += k + " x" + strconv.Itoa(v) + "\n"
	}
	ret = append(ret, foodsStr)

	var makablesStr, unmakablesStr string
	if strings.Contains(text, "サラダ") {
		makablesStr, unmakablesStr = dres.GetCookResultString(c.Salad)
	} else if strings.Contains(text, "カレー") {
		makablesStr, unmakablesStr = dres.GetCookResultString(c.Curry)
	} else if strings.Contains(text, "デザート") {
		makablesStr, unmakablesStr = dres.GetCookResultString(c.Desert)
	} else {
		makables, unmakables := dres.GetCookResultString(c.Salad)
		makablesStr += "\nサラダの" + makables
		unmakablesStr += "\nサラダの" + unmakables
		makables, unmakables = dres.GetCookResultString(c.Curry)
		makablesStr += "\nカレーの" + makables
		unmakablesStr += "\nカレーの" + unmakables
		makables, unmakables = dres.GetCookResultString(c.Desert)
		makablesStr += "\nデザートの" + makables
		unmakablesStr += "\nデザートの" + unmakables
	}
	ret = append(ret, makablesStr, unmakablesStr)

	return ret, nil
}

func (c *Client) OCR(ctx context.Context, img *Image) (*DetectResult, error) {
	// Vision AIに読み込ませる準備
	visionImg, err := vision.NewImageFromReader(img.Bytes)
	if err != nil {
		return nil, fmt.Errorf("load file failed: %w", err)
	}

	// 実行
	annotations, err := c.Vision.DetectTexts(ctx, visionImg, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("execute ocr failed: %w", err)
	}

	// err = img.UpdateImage(annotations)
	// if err != nil {
	// 	return fmt.Errorf("NewImage failed: %w", err)
	// }
	dresult := NewDetectedResult(img, annotations)

	return dresult, nil
}

func DownloadImage(url, token string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, fmt.Errorf("access %s failed: %w", url, err)
	}
	return resp, nil
}

func LoadJsonConfig(path string, client *Client) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file failed: %w", err)
	}
	defer file.Close()
	jsonData, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("load file failed: %w", err)
	}
	err = json.Unmarshal(jsonData, client)
	if err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}
	return nil
}
