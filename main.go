package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const windowSize = 10

var apiEndpoints = map[string]string{
	"p": "http://20.244.56.144/evaluation-service/primes",
	"f": "http://20.244.56.144/evaluation-service/fibo",
	"e": "http://20.244.56.144/evaluation-service/even",
	"r": "http://20.244.56.144/evaluation-service/rand",
}

const clientID = "23a5f0ce-f938-4c53-9f0e-59b7eedbb73a"
const clientSecret = "ZurwgmTjXFrwuKMs"

type NumberStore struct {
	mu      sync.Mutex
	numbers []int
}

type ApiResponse struct {
	Numbers []int `json:"numbers"`
}

var store = &NumberStore{}

func fetchNumbers(url string) ([]int, error) {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	const accessToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJNYXBDbGFpbXMiOnsiZXhwIjoxNzQzNjAzODc5LCJpYXQiOjE3NDM2MDM1NzksImlzcyI6IkFmZm9yZG1lZCIsImp0aSI6IjIzYTVmMGNlLWY5MzgtNGM1My05ZjBlLTU5YjdlZWRiYjczYSIsInN1YiI6IjIyMDUxODRAa2lpdC5hYy5pbiJ9LCJlbWFpbCI6IjIyMDUxODRAa2lpdC5hYy5pbiIsIm5hbWUiOiJhbmtpdCBiYXNhayIsInJvbGxObyI6IjIyMDUxODQiLCJhY2Nlc3NDb2RlIjoibndwd3JaIiwiY2xpZW50SUQiOiIyM2E1ZjBjZS1mOTM4LTRjNTMtOWYwZS01OWI3ZWVkYmI3M2EiLCJjbGllbnRTZWNyZXQiOiJadXJ3Z21UalhGcnd1S01zIn0.RCz4QAJorPGXsWAuu-opnQ0RtYlBuNT7v4qOZmLr7e0"

	req.Header.Set("Authorization", "Bearer "+accessToken)
	log.Println("Request Headers:", req.Header)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("Error decoding response:", err)
		return nil, err
	}

	return result.Numbers, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (s *NumberStore) updateWindow(newNumbers []int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	numberSet := make(map[int]bool)
	var updatedNumbers []int

	for _, num := range s.numbers {
		numberSet[num] = true
		updatedNumbers = append(updatedNumbers, num)
	}

	for _, num := range newNumbers {
		if !numberSet[num] {
			updatedNumbers = append(updatedNumbers, num)
			numberSet[num] = true
		}
	}

	if len(updatedNumbers) > windowSize {
		updatedNumbers = updatedNumbers[len(updatedNumbers)-windowSize:]
	}

	s.numbers = updatedNumbers
}

func (s *NumberStore) calculateAverage() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.numbers) == 0 {
		return 0.0
	}

	sum := 0
	for _, num := range s.numbers {
		sum += num
	}
	return float64(sum) / float64(len(s.numbers))
}

func getNumbersHandler(c *gin.Context) {
	numberID := c.Param("numberid")
	apiURL, valid := apiEndpoints[numberID]
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid number ID"})
		return
	}

	newNumbers, err := fetchNumbers(apiURL)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to fetch numbers"})
		return
	}

	prevState := make([]int, len(store.numbers))
	copy(prevState, store.numbers)

	store.updateWindow(newNumbers)

	avg := store.calculateAverage()

	c.JSON(http.StatusOK, gin.H{
		"windowPrevState": prevState,
		"windowCurrState": store.numbers,
		"numbers":         newNumbers,
		"avg":             avg,
	})
}

func main() {
	router := gin.Default()
	router.GET("/numbers/:numberid", getNumbersHandler)

	log.Println("Server is running on port 9876...")
	router.Run(":9876") // Start server
}
