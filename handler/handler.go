package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/BITGIN/bitgin-api-docs/config"
	"github.com/BITGIN/bitgin-api-docs/model"
	"github.com/labstack/echo/v4"
)

func sign(payload string) string {
	hash := hmac.New(sha256.New, []byte(config.Secret))
	hash.Write([]byte(payload))
	signature := hex.EncodeToString(hash.Sum(nil))
	return signature
}

func randFunc() string {
	rand.Seed(time.Now().Unix())
	// 2^32
	x := rand.Int63n(int64(math.Pow(2, 32)))
	return fmt.Sprintf("%08x", x)
}

func FaasPayHandler(c echo.Context) error {
	body := model.RequestBodyPay{}
	if err := c.Bind(&body); err != nil {
		log.Println("error: ", err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}

	data, _ := json.Marshal(&body)

	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, " ", "")
	dataStr = strings.ReplaceAll(dataStr, "\n", "")
	dataStr = strings.ReplaceAll(dataStr, "\t", "")

	method := http.MethodPost
	path := "/v1/faas/pay"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randFunc()

	payload := fmt.Sprintf("%s%s%s%s%s", method, path, nonce, timestamp, dataStr)
	fmt.Println("payload: ", payload)

	signature := sign(payload)
	fmt.Println("signature: ", signature)

	u, _ := url.Parse(config.Frontend_Endpoint)
	q := u.Query()
	q.Add("key", config.Key)
	q.Add("sign", signature)
	q.Add("nonce", nonce)
	q.Add("timestamp", timestamp)
	q.Add("body", base64.RawURLEncoding.EncodeToString(data))
	u.RawQuery = q.Encode()

	log.Println("url: ", u.String())

	res := model.ResponseBodyPay{
		URL: u.String(),
	}
	return c.JSON(http.StatusOK, res)
}

func FaasReceiptHandler(c echo.Context) error {

	body := model.RequestBodyReceipt{}
	if err := c.Bind(&body); err != nil {
		log.Println("error: ", err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}

	var dataBytes []byte
	var dataStr string
	if c.Request().Body != nil {
		data, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		dataBytes = data

		c.Request().Body = ioutil.NopCloser(bytes.NewReader(data))
		dataStr = string(data)
		dataStr = strings.ReplaceAll(dataStr, " ", "")
		dataStr = strings.ReplaceAll(dataStr, "\n", "")
		dataStr = strings.ReplaceAll(dataStr, "\t", "")
	}

	method := http.MethodGet
	path := "/v1/faas/receipt"
	nonce := randFunc()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	payload := fmt.Sprintf("%s%s%s%s%s", method, path, nonce, timestamp, dataStr)
	fmt.Println("payload: ", payload)

	signature := sign(payload)
	fmt.Println("signature: ", signature)

	log.Printf("key:%s&sign:%s&nonce:%s&timestamp:%s", config.Key, signature, nonce, timestamp)

	req, err := http.NewRequest(http.MethodGet, config.Backend_Endpoint+path, bytes.NewBuffer(dataBytes))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	reqHeader := map[string]string{
		"BG-API-KEY":       config.Key,
		"BG-API-SIGN":      signature,
		"BG-API-NONCE":     nonce,
		"BG-API-TIMESTAMP": timestamp,
		"Content-Type":     "application/json; charset=UTF-8",
	}

	for k, v := range reqHeader {
		req.Header.Set(k, v)
	}

	reqParams := map[string]string{}

	if body.OrderID != nil {
		reqParams["order_id"] = *body.OrderID
	}
	if body.Currency != nil {
		reqParams["currency"] = *body.Currency
	}
	if body.StartDateUnix != nil {
		reqParams["start_date"] = strconv.FormatInt(*body.StartDateUnix, 10)
	}
	if body.EndDateUnix != nil {
		reqParams["end_date"] = strconv.FormatInt(*body.EndDateUnix, 10)
	}
	if body.Limit > 0 {
		reqParams["limit"] = strconv.Itoa(body.Limit)
	}
	if body.Offset < 0 {
		reqParams["offset"] = strconv.Itoa(body.Offset)
	}

	query := req.URL.Query()
	for k, v := range reqParams {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	var apiRes model.ApiResponseFaasReceipt
	err = json.Unmarshal(respbody, &apiRes)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.JSON(resp.StatusCode, apiRes)
}

func MineQueryAddressesHandler(c echo.Context) error {

	body := model.MineCheckBitginAddressRequest{}
	if err := c.Bind(&body); err != nil {
		log.Println("error: ", err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}
	data, _ := json.Marshal(body)

	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, " ", "")
	dataStr = strings.ReplaceAll(dataStr, "\n", "")
	dataStr = strings.ReplaceAll(dataStr, "\t", "")

	method := http.MethodPost
	path := "/v1/mine/query"
	nonce := randFunc()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	payload := fmt.Sprintf("%s%s%s%s%s", method, path, nonce, timestamp, dataStr)

	fmt.Println("payload: ", payload)

	signature := sign(payload)
	fmt.Println("signature: ", signature)

	log.Printf("key:%s&sign:%s&nonce:%s&timestamp:%s", config.Key, signature, nonce, timestamp)

	req, err := http.NewRequest("POST", config.Backend_Endpoint+path, bytes.NewBuffer(data))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	reqHeader := map[string]string{
		"BG-API-KEY":       config.Key,
		"BG-API-SIGN":      signature,
		"BG-API-NONCE":     nonce,
		"BG-API-TIMESTAMP": timestamp,
		"Content-Type":     "application/json; charset=UTF-8",
	}

	for k, v := range reqHeader {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	var apiRes model.ApiResponseMineQuery
	err = json.Unmarshal(respbody, &apiRes)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.JSON(resp.StatusCode, apiRes)
}

func MineShareHandler(c echo.Context) error {

	body := model.MineShareReq{}
	if err := c.Bind(&body); err != nil {
		log.Println("error: ", err.Error())
		return c.String(http.StatusBadRequest, err.Error())
	}
	data, _ := json.Marshal(body)

	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, " ", "")
	dataStr = strings.ReplaceAll(dataStr, "\n", "")
	dataStr = strings.ReplaceAll(dataStr, "\t", "")

	method := http.MethodPost
	path := "/v1/mine/share"
	nonce := randFunc()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	payload := fmt.Sprintf("%s%s%s%s%s", method, path, nonce, timestamp, dataStr)
	fmt.Println("payload: ", payload)

	signature := sign(payload)
	fmt.Println("signature: ", signature)

	log.Printf("key:%s&sign:%s&nonce:%s&timestamp:%s", config.Key, signature, nonce, timestamp)

	req, err := http.NewRequest("POST", config.Backend_Endpoint+path, bytes.NewBuffer(data))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	reqHeader := map[string]string{
		"BG-API-KEY":       config.Key,
		"BG-API-SIGN":      signature,
		"BG-API-NONCE":     nonce,
		"BG-API-TIMESTAMP": timestamp,
		"Content-Type":     "application/json; charset=UTF-8",
	}

	for k, v := range reqHeader {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	var apiRes model.ApiResponseMineShare
	err = json.Unmarshal(respbody, &apiRes)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.JSON(resp.StatusCode, apiRes)
}
