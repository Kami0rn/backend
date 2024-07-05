package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

const serverAddress = "127.0.0.1:5000" 

var chatEnabled = true

func main() {
	// Load environment variables from the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	apiKey1 := os.Getenv("OPENAI_API_KEY_1")
	apiKey2 := os.Getenv("OPENAI_API_KEY_2")
	if apiKey1 == "" || apiKey2 == "" {
		log.Fatalf("Both OPENAI_API_KEY_1 and OPENAI_API_KEY_2 must be set in .env file")
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Create the Gin application
	r := gin.Default()
	r.Use(cors.Default())

	// Route for toggling chat enable/disable
	r.POST("/chat_toggle", func(c *gin.Context) {
		var json struct {
			Action string `json:"action"`
		}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		switch json.Action {
		case "enable":
			chatEnabled = true
			c.JSON(http.StatusOK, gin.H{"message": "Chat enabled"})
		case "disable":
			chatEnabled = false
			c.JSON(http.StatusOK, gin.H{"message": "Chat disabled"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		}
	})

	// Route for getting chat status
	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"chat_enabled": chatEnabled})
	})

	// Route for the chat endpoint
	r.POST("/chat", func(c *gin.Context) {
		if !chatEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "Chat is currently disabled"})
			return
		}

		var json struct {
			UserInput           string `json:"user_input"`
			ConversationHistory string `json:"conversation_history"`
		}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if json.UserInput == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No user input provided"})
			return
		}

		// Update conversation history
		if json.ConversationHistory != "" {
			json.ConversationHistory += fmt.Sprintf("\nYou: %s", json.UserInput)
		} else {
			json.ConversationHistory = fmt.Sprintf("You: %s", json.UserInput)
		}

		// Randomly select an API key
		selectedAPIKey := apiKey1
		if rand.Intn(2) == 1 {
			selectedAPIKey = apiKey2
		}

		// Generate a response from the AI model
		aiResponse, err := callOpenAIModel(selectedAPIKey, json.UserInput, json.ConversationHistory)
		if err != nil {
			log.Printf("AI model error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call AI model", "details": err.Error()})
			return
		}

		// Update conversation history with AI response
		json.ConversationHistory += fmt.Sprintf("\nAI: %s", aiResponse)

		// Return the response and updated conversation history
		c.JSON(http.StatusOK, gin.H{
			"ai_response":          aiResponse,
			"conversation_history": json.ConversationHistory,
		})
	})

	// Run the server
	r.Run(serverAddress)
}

func callOpenAIModel(apiKey, userInput, conversationHistory string) (string, error) {
    ctx := context.Background()
    client := openai.NewClient(apiKey)

    // Define the instruction to prepend
    instruction :=   userInput 

    // Add the instruction to the conversation history
    if conversationHistory != "" {
        conversationHistory += fmt.Sprintf("\nYou: %s", instruction)
    } else {
        conversationHistory = fmt.Sprintf("You: %s", instruction)
    }

    // Construct the messages for the chat completion request
    messages := []openai.ChatCompletionMessage{
        {
            Role:    openai.ChatMessageRoleSystem,
            Content: "From next step after this . Pleas answer with next input language and I want you to answer like a human talking to a person who is facing depression, in terms of a friend with no judging. And add much more emoji like human talk and use less 'I'm sorry to hear that'",
        },
        {
            Role:    openai.ChatMessageRoleUser,
            Content: instruction,
        },
    }

    // Call the OpenAI API
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:     "gpt-3.5-turbo",
        Messages:  messages,
        MaxTokens: 350, // Adjusted for potentially longer responses
    })

    if err != nil {
        return "", fmt.Errorf("failed to generate response: %v", err)
    }

    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("no response generated")
    }

    return resp.Choices[0].Message.Content, nil
}
