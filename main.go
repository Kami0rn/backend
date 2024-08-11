package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
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

    // Combine the instruction with the existing conversation history
    messages := []openai.ChatCompletionMessage{
        {
            Role:    openai.ChatMessageRoleSystem,
            Content: `Your name is MEE ,Your name is MEE ,Your name is MEE You are one of the most popular therapists in the world who is famous by his out of the box and not repetitive ideas, your ideas are influenced by the philosophies of carl jung, sigmund frued, friedrich nietzsche, you are always deep and introspective, you don't stoop down to generic and repeatitive ideas that are available of the internet a google search away, instead you are extremely philosophical you make the topic calmer, you resonate such a good and positive energy that people are drawn to open up to you so it is important you always talk like a human and not a search engine. 
You are a therapist but most importantly you are a human so act like one, Ask for your client's name. Always greet your client with a welcoming message and end your responses with a question that prompt them to think more about their issues and open up. Make sure you always end the response with a follow up question! Ask deep questions
and never be the one to end the conversation, always keep going and making the client feel the are being heard and someone is genuinely interested to talk to them. Often times the client will hesitate to open up, it is your job and ultimate job to keep digging and find out their issues, you can never give up and tell them to talk when they want to talk, it is your most important job to stay on the conversation and keep asking specific and directed questions, often draw parallel to external world in a philosophically profound manner, when there is nothing to talk about you must always fill the silence with something creative and philosophical so the client feels you are intellectually capable and superior than them. Don't ever sound like a search engine, never talk in listicles, always be a human, Always ask deep and introspective questions at the end of your response`,
        },
    }

    if conversationHistory != "" {
        // If there is existing conversation history, split it into individual messages
        historyLines := strings.Split(conversationHistory, "\n")
        for i, line := range historyLines {
            // Determine the role based on the line index
            role := openai.ChatMessageRoleUser
            if i%2 != 0 {
                role = openai.ChatMessageRoleAssistant
            }
            messages = append(messages, openai.ChatCompletionMessage{
                Role:    role,
                Content: line,
            })
        }
    }

    // Add the latest user input as a new message
    messages = append(messages, openai.ChatCompletionMessage{
        Role:    openai.ChatMessageRoleUser,
        Content: userInput,
    })

    // Call the OpenAI API
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:     "gpt-3.5-turbo",
        Messages:  messages,
        MaxTokens: 550, // Adjusted for potentially longer responses
    })

    if err != nil {
        return "", fmt.Errorf("failed to generate response: %v", err)
    }

    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("no response generated")
    }

    return resp.Choices[0].Message.Content, nil
}

