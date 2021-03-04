package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// Subscriptions holds information for push notification subscription
type Subscriptions struct {
	publicKey         string
	privateKey        string
	subscriptions     []webpush.Subscription
	subscriptionsFile string
}

// NewSubscriptions reads files or creates them if necessary
func NewSubscriptions(privateKeyFile, publicKeyFile, subscriptionsFile string) (*Subscriptions, error) {
	fPub, err := os.OpenFile(publicKeyFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer fPub.Close()
	pubKeyBytes, err := io.ReadAll(fPub)
	if err != nil {
		return nil, fmt.Errorf("cannot read public key %s: %v", publicKeyFile, err)
	}
	pubKey := string(pubKeyBytes)

	fPrivate, err := os.OpenFile(privateKeyFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer fPrivate.Close()
	privateKeyBytes, err := io.ReadAll(fPrivate)
	if err != nil {
		return nil, fmt.Errorf("cannot read private key %s: %v", publicKeyFile, err)
	}
	privateKey := string(privateKeyBytes)

	newKeys := false

	// generate key pair if needed
	if len(pubKey) == 0 && len(privateKey) == 0 {
		log.Infof("generating new key pair in '%s' (private) and '%s' (public)", privateKeyFile, publicKeyFile)
		privateKey, pubKey, err = webpush.GenerateVAPIDKeys()
		if err != nil {
			return nil, fmt.Errorf("cannot generate key pair: %v", err)
		}
		_, err := fPrivate.WriteString(privateKey)
		if err != nil {
			return nil, fmt.Errorf("cannot save private key to %s: %v", privateKeyFile, err)
		}
		_, err = fPub.WriteString(pubKey)
		if err != nil {
			return nil, fmt.Errorf("cannot save public key to %s: %v", publicKeyFile, err)
		}
		newKeys = true
	} else if len(pubKey) == 0 {
		return nil, fmt.Errorf("public key file '%s' is empty", publicKeyFile)
	} else if len(privateKey) == 0 {
		return nil, fmt.Errorf("private key file '%s' is empty", publicKeyFile)
	}

	subFile, err := os.OpenFile(subscriptionsFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	subs := Subscriptions{
		privateKey:        privateKey,
		publicKey:         pubKey,
		subscriptions:     []webpush.Subscription{},
		subscriptionsFile: subscriptionsFile,
	}

	if newKeys {
		log.Info("clearing subscriptions...")
		subFile.Truncate(0)
		subFile.Seek(0, 0)
	} else {
		subBytes, err := io.ReadAll(subFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read subscriptions %s: %v", subscriptionsFile, err)
		}
		for _, sub := range strings.Split(string(subBytes), "\n") {
			subs.Add([]byte(sub))
		}
	}

	return &subs, nil

}

// Add adds a subscription
func (s *Subscriptions) Add(jsonBytes []byte) error {
	sub := webpush.Subscription{}
	err := json.Unmarshal([]byte(jsonBytes), &sub)
	if err != nil {
		return fmt.Errorf("invalid subscription body: %v", err)
	}

	f, err := os.OpenFile(s.subscriptionsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	f.Write(jsonBytes)
	f.WriteString("\n")

	s.subscriptions = append(s.subscriptions, sub)
	return nil
}

// Send sends push notification to all subscribers
func (s Subscriptions) Send(message string) {

	worker := func(jobs <-chan webpush.Subscription, wg *sync.WaitGroup) {
		defer wg.Done()
		for m := range jobs {
			// Send Notification
			resp, err := webpush.SendNotification([]byte(message), &m, &webpush.Options{
				Subscriber:      "info@mmotcw.club", // Do not include "mailto:"
				VAPIDPublicKey:  s.publicKey,
				VAPIDPrivateKey: s.privateKey,
				TTL:             30,
			})
			if err != nil {
				log.Errorf("cannot send push notification: %v", err)
			}
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				respBody, _ := io.ReadAll(resp.Body)
				log.Warnf("push notification response: %s (status %d)", string(respBody), resp.StatusCode)
			}
		}
	}

	var wg sync.WaitGroup
	var jobs chan webpush.Subscription = make(chan webpush.Subscription)

	numCores := runtime.NumCPU()
	wg.Add(numCores)
	for w := 0; w < numCores; w++ {
		go worker(jobs, &wg)
	}

	for _, sub := range s.subscriptions {
		jobs <- sub
	}
	close(jobs)
	wg.Wait()
}

func subscribe(s *Subscriptions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("error reading http body:", err)
			httpError(w, http.StatusInternalServerError)
			return
		}

		err = s.Add(data)
		if err != nil {
			log.Error("cannot process subscription: ", err)
			httpError(w, http.StatusBadRequest)
			return
		}
		log.Info("registered push notification subscription")

		fmt.Fprint(w, "ok")
	}
}
