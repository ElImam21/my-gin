package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Ollama API request body
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

func streamDeepseek(prompt string, writer http.ResponseWriter) error {
	requestBody, err := json.Marshal(OllamaRequest{
		Model:  "deepseek-r1:latest",
		Prompt: prompt,
	})
	if err != nil {
		return fmt.Errorf("error marshalling request: %v", err)
	}

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("error calling deepseek: %v", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var chunk map[string]interface{}
		if err := decoder.Decode(&chunk); err != nil {
			break
		}
		if part, ok := chunk["response"].(string); ok {
			writer.Write([]byte(part))
			writer.(http.Flusher).Flush()
		}
	}
	return nil
}

func main() {
	router := gin.Default()

	router.POST("/ask", func(c *gin.Context) {
		var req struct {
			Message string `json:"message"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")

		if err := streamDeepseek(req.Message, c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stream deepseek", "details": err.Error()})
			return
		}
	})

	router.Run(":8080")
}
