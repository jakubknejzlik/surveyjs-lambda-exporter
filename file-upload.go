package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func uploadCSVFile(ctx context.Context, content []byte, filename string) (fileID string, err error) {
	fileUploadURL := os.Getenv("FILE_UPLOAD_URL")
	if fileUploadURL == "" {
		err = fmt.Errorf("Missing required FILE_UPLOAD_URL envvar")
		return
	}

	values := map[string]interface{}{
		"filename":    filename,
		"contentType": "text/csv",
		"size":        len(content),
	}

	jsonValue, err := json.Marshal(values)
	if err != nil {
		return
	}

	resp, err := http.Post(fileUploadURL, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var res struct {
		ID        string
		UploadURL string
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return
	}
	req, err := http.NewRequest("PUT", res.UploadURL, bytes.NewBuffer(content))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/csv")

	client := &http.Client{}
	uploadResp, err := client.Do(req)
	if err != nil {
		return
	}
	if uploadResp.StatusCode != 200 {
		err = fmt.Errorf("Unexpected upload response code %d", uploadResp.StatusCode)
		return
	}

	fileID = res.ID

	return
}
