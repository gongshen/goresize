package main

import (
	"code.google.com/p/tcgl/redis"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/bmizerany/pat"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
)

// The maximal size for an image is 5MB
const maxSize = 5 * (1 << 20)

// The directory for caching files
var directory string

// The connection to redis
var connection *redis.Database

func urlStatus(uri string) error {
	res := connection.Command("hexists", "img/"+uri, "created_at")
	if !res.IsOK() {
		return res.Error()
	}

	ok, err := res.ValueAsBool()
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("Invalid URL")
	}

	return nil
}

// Generate a key for cache from a string
func generateKeyForCache(s string) string {
	h := sha1.New()
	io.WriteString(h, s)
	key := h.Sum(nil)

	// Use 3 levels of hasing to avoid having too many files in the same directory
	return fmt.Sprintf("%s/%x/%x/%x/%x", directory, key[0:1], key[1:2], key[2:3], key[3:])
}

// Fetch image from cache
func fetchImageFromCache(uri string) (contentType string, body []byte, ok bool) {
	ok = false

	res := connection.Command("hget", "img/"+uri, "type")
	if !res.IsOK() {
		return
	}
	contentType = res.String()

	filename := generateKeyForCache(uri)
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	ok = true
	return
}

// Save the body and the content-type header in cache
func saveImageInCache(uri string, contentType string, body []byte) {
	filename := generateKeyForCache(uri)
	dirname := path.Dir(filename)
	err := os.MkdirAll(dirname, 0755)
	if err != nil {
		return
	}

	// Save the body on disk
	err = ioutil.WriteFile(filename, body, 0644)
	if err != nil {
		log.Printf("Error while writing %s\n", filename)
		return
	}

	// And other infos in redis
	connection.Command("hset", "img/"+uri, "type", contentType)
}

// Fetch the image from the distant server
func fetchImageFromServer(uri string) (contentType string, body []byte, err error) {
	res, err := http.Get(uri)
	if err != nil {
		return
	}
	if res.StatusCode != 200 {
		log.Printf("Status code of %s is: %d\n", uri, res.StatusCode)
		err = errors.New("Unexpected status code")
		return
	}

	defer res.Body.Close()
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	// TODO keep errors in cache with a small TTL
	if res.ContentLength > maxSize {
		log.Printf("Exceeded max size for %s: %d\n", uri, res.ContentLength)
		err = errors.New("Exceeded max size")
		return
	}
	contentType = res.Header.Get("Content-Type")
	if contentType[0:5] != "image" {
		log.Printf("%s has an invalid content-type: %s\n", uri, contentType)
		err = errors.New("Invalid content-type")
		return
	}
	log.Printf("Fetch %s (%s)\n", uri, contentType)

	if urlStatus(uri) == nil {
		go saveImageInCache(uri, contentType, body)
	}
	return
}

// Fetch image from cache if available, or from the server
func fetchImage(uri string) (contentType string, body []byte, err error) {
	err = urlStatus(uri)
	if err != nil {
		return
	}

	contentType, body, ok := fetchImageFromCache(uri)
	if !ok {
		contentType, body, err = fetchImageFromServer(uri)
	}

	return
}

// Receive an HTTP request for an image and respond with it
func Img(w http.ResponseWriter, r *http.Request) {
	encoded_url := r.URL.Query().Get(":encoded_url")
	chars, err := hex.DecodeString(encoded_url)
	if err != nil {
		log.Printf("Invalid URL %s\n", encoded_url)
		http.Error(w, "Invalid parameters", 400)
		return
	}
	uri := string(chars)

	contentType, body, err := fetchImage(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Add("Content-Type", contentType)
	w.Write(body)
}

// Returns 200 OK if the server is running (for monitoring)
func Status(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func main() {
	// Parse the command-line
	var addr string
	var logs string
	var conn string
	flag.StringVar(&addr, "a", "127.0.0.1:8000", "Bind to this address:port")
	flag.StringVar(&logs, "l", "-", "Use this file for logs")
	flag.StringVar(&conn, "-r", "localhost:6379/0", "The redis database to use for caching meta")
	flag.StringVar(&directory, "d", "cache", "The directory for the caching files")
	flag.Parse()

	// Logging
	if logs != "-" {
		f, err := os.OpenFile(logs, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatal("OpenFile: ", err)
		}
		syscall.Dup2(int(f.Fd()), int(os.Stdout.Fd()))
		syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
	}

	// Redis
	parts := strings.Split(conn, "/")
	host := parts[0]
	db := 0
	if len(parts) >= 2 {
		db, _ = strconv.Atoi(parts[1])
	}
	cfg := redis.Configuration{Database: db, Address: host, PoolSize: 4, LogCommands: false}
	connection = redis.Connect(cfg)

	// Routing
	m := pat.New()
	m.Get("/status", http.HandlerFunc(Status))
	m.Get("/img/:encoded_url", http.HandlerFunc(Img))
	http.Handle("/", m)

	// Start the HTTP server
	log.Printf("Listening on http://%s/\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
