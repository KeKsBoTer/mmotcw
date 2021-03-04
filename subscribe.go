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

func NewSubscriptions(privateKeyFile, publicKeyFile, subscriptionsFile string) (*Subscriptions, error) {
	fpub, err := os.OpenFile(publicKeyFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer fpub.Close()
	pubKeyBytes, err := io.ReadAll(fpub)
	if err != nil {
		return nil, fmt.Errorf("cannot read public key %s: %v", publicKeyFile, err)
	}
	pubKey := string(pubKeyBytes)

	fprivate, err := os.OpenFile(privateKeyFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer fprivate.Close()
	privateKeyBytes, err := io.ReadAll(fprivate)
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
		_, err := fprivate.WriteString(privateKey)
		if err != nil {
			return nil, fmt.Errorf("cannot save private key to %s: %v", privateKeyFile, err)
		}
		_, err = fpub.WriteString(pubKey)
		if err != nil {
			return nil, fmt.Errorf("cannot save public key to %s: %v", publicKeyFile, err)
		}
		newKeys = true
	} else if len(pubKey) == 0 {
		return nil, fmt.Errorf("public key file '%s' is empty", publicKeyFile)
	} else if len(privateKey) == 0 {
		return nil, fmt.Errorf("private key file '%s' is empty", publicKeyFile)
	}

	subf, err := os.OpenFile(subscriptionsFile, os.O_RDWR|os.O_CREATE, 0666)
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
		subf.Truncate(0)
		subf.Seek(0, 0)
	} else {
		subBytes, err := io.ReadAll(subf)
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

	f, err := os.OpenFile(s.subscriptionsFile, os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	f.WriteString("\n")
	f.Write(jsonBytes)

	s.subscriptions = append(s.subscriptions, sub)
	return nil
}

// Send sents push notification to all subscribers
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
			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				log.Warnf("push notification response: %s", string(respBody))
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
		data, err := io.ReadAll(r.Response.Body)
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

		fmt.Fprint(w, "ok")
		w.WriteHeader(http.StatusOK)
	}
}
