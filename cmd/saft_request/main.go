package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Response struct {
	Url string `json:"url"`
}

func main() {
	for i := 0; i < 6; i++ {
		log.Println("Trying to get the link")
		if requestLink() {
			log.Println("link sent")
			return
		}

		time.Sleep(5 * time.Second)
	}

	log.Println("all retries exhausted")
}

func requestLink() bool {
	now := time.Now()
	url := fmt.Sprintf(
		"https://%s.app.invoicexpress.com/api/export_saft.json?month=%s&years=%s&api_key=%s",
		os.Getenv("ACCOUNT_NAME"),
		now.Format("01"),
		now.Format("2006"),
		os.Getenv("API_KEY"),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %s", err)
	}

	req.Header.Add("accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Failed to send request: %s", err)
	}

	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		SendSAFTLink(res)

		return true
	}

	if res.StatusCode != http.StatusAccepted {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("unable to read body: %s", err)
		}

		log.Fatalf("unexpected error with status code %d: %s", res.StatusCode, body)
	}

	return false
}

func SendSAFTLink(res *http.Response) {
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("unable to read body: %s", err)
	}

	var result Response
	if err = json.Unmarshal(body, &result); err != nil {
		log.Fatalf("unable to read body: %s", err)
	}

	url := result.Url
	res, err = http.Get(url)
	if err != nil {
		log.Fatalf("unable to send request: %s", err)
	}

	now := time.Now()
	filePath := fmt.Sprintf("./SAFT_%s%s.zip", now.Format("01"), now.Format("2006"))
	out, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("unable to save SAF-T file: %s", err)
	}
	defer out.Close()
	io.Copy(out, res.Body)

	from := mail.NewEmail(os.Getenv("EMAIL_FROM_NAME"), os.Getenv("EMAIL_FROM"))
	subject := os.Getenv("EMAIL_SUBJECT")
	to := mail.NewEmail(os.Getenv("EMAIL_TO_NAME"), os.Getenv("EMAIL_TO"))

	plainTextContent := os.Getenv("EMAIL_CONTENT")
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, plainTextContent)
	client := sendgrid.NewSendClient(os.Getenv("EMAIL_API_KEY"))
	_, err = client.Send(message)

	if err != nil {
		log.Fatalf("unable to send email with SAF-T file: %s", err)
	}
}
