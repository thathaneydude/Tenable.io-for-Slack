package main

//
//import (
//	"bytes"
//	"fmt"
//	"log"
//	"net/http"
//	"time"
//)
//
//type Event struct {
//	ID          string      `json:"id"`
//	Action      string      `json:"action"`
//	Crud        string      `json:"crud"`
//	IsFailure   bool        `json:"is_failure"`
//	Received    time.Time   `json:"received"`
//	Description interface{} `json:"description"`
//	Actor       struct {
//		ID   string `json:"id"`
//		Name string `json:"name"`
//	} `json:"actor"`
//	IsAnonymous interface{} `json:"is_anonymous"`
//	Target      struct {
//		ID   string `json:"id"`
//		Name string `json:"name"`
//		Type string `json:"type"`
//	} `json:"target"`
//	Fields []interface{} `json:"fields"`
//}
//
//type AuditLogResponse struct {
//	Events     []Event `json:"events"`
//	Pagination struct {
//		Total int `json:"total"`
//		Limit int `json:"limit"`
//	} `json:"pagination"`
//}
//
//type TenableIOClient struct {
//	client    *http.Client
//	basePath  string
//	accessKey string
//	secretKey string
//}
//
//func NewTenableIOClient(accessKey string, secretKey string) TenableIOClient {
//	client := &http.Client{}
//	tio := &TenableIOClient{
//		client,
//		baseTenablePath,
//		accessKey,
//		secretKey,
//	}
//	return *tio
//}
//
//func (tio *TenableIOClient) Do(req *http.Request) http.Response {
//	req.Header.Set("X-ApiKeys", fmt.Sprintf("accessKey=%v; secretKey=%v;", tio.accessKey, tio.secretKey))
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("User-Agent", "GoTenable")
//	//log.Printf("Requesting \"%v\": %v\n", req.URL, req.Body)
//	resp, err := tio.client.Do(req)
//	if err != nil {
//		log.Printf("Unable to run request: %v\n", err)
//	}
//
//	return *resp
//}
//
//func (tio *TenableIOClient) NewRequest(method string, endpoint string, body []byte) *http.Request {
//	fullUrl := fmt.Sprintf("%v/%v", baseTenablePath, endpoint)
//	req, err := http.NewRequest(method, fullUrl, bytes.NewBuffer(body))
//	if err != nil {
//		log.Printf("Unable to build request [%v] %v request\n", method, endpoint)
//	}
//	return req
//}
//
