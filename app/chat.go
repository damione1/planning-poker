package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mannders00/deploysolo/utils"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterChat(se *core.ServeEvent) error {
	g := se.Router.Group("/app")
	g.Bind(utils.RequirePayment())

	g.GET("/chat", utils.RenderView("chat.html", nil)).Bind(apis.RequireAuth())
	g.POST("/chat", chatHandler)

	return nil
}

// ChatHandler is an HTTP handler that takes takes the list of messages as a FormValue
// Extracts the message history from HTML using goquery, then returns an html chunk
// for htmx to render at the end of the message history. The state of the app
// is stored in the browser's DOM, which is a great example of HATEAOS.
func chatHandler(e *core.RequestEvent) error {
	chatHTML := e.Request.FormValue("chat-include") // This receives the entire chat HTML as a string
	fmt.Println(chatHTML)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(chatHTML))
	if err != nil {
		return err
	}

	// Extract chat conversation as array direcrtly from HTML markup sent with htmx
	// Amazing example of HATEAOS storing relevant application state in browser.
	var conversation []Message

	doc.Find(".user-chat-message, .assistant-chat-message").Each(func(i int, s *goquery.Selection) {
		role := "user" // Default role
		if s.HasClass("assistant-chat-message") {
			role = "assistant"
		}

		message := Message{
			Role:    role,
			Content: strings.TrimSpace(s.Text()),
		}
		conversation = append(conversation, message)
	})

	cfg := e.App.Store().Get("cfg").(*utils.Config)
	response, err := callChatGPT(conversation, cfg)
	if err != nil {
		return err
	}

	htmlFragment := `<div class="assistant-chat-message align-self-start bg-white text-dark border rounded p-2 mb-2 shadow-sm"
style="max-width: 70%%;">%s</div>`

	htmlResponse := fmt.Sprintf(htmlFragment, response)

	return e.HTML(http.StatusOK, htmlResponse)

}

// Message represents a single message in the conversation with its role and content.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request body for the ChatGPT API.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Choice represents each choice returned by the API, containing the message and other metadata.
type Choice struct {
	Index        int              `json:"index"`
	Message      Message          `json:"message"`
	Logprobs     *json.RawMessage `json:"logprobs"`
	FinishReason string           `json:"finish_reason"`
}

// ChatResponse is the expected structure of the response from the ChatGPT API.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// createChatGPTRequest prepares the request data for the API call.
func createChatGPTRequest(conversation []Message) ChatRequest {
	prompt := `
	You are talking to a general conversational model that is suited for general questions but also
	Is specialized in assisting with a product named DeploySolo
	DeploySolo is the SaaS boilerplate for Go Engineers and Indie Hackers. It outlines a set of practices to develop modern, interactive web apps as easily as possible.
	It is built with PocketBase for the backend, and htmx for the front end, making it easy to deploy with a single executable, but also create reactive
	experiences wihout introducing front end frameworks. Don't fabricate specific code, just answer abstractly, and redirect users to official documentation if needed.
	`

	// Add any system initialization messages if required.
	systemMessage := Message{
		Role:    "system",
		Content: prompt,
	}

	return ChatRequest{
		Model:    "gpt-3.5-turbo",
		Messages: append([]Message{systemMessage}, conversation...),
	}
}

// callChatGPT sends the structured conversation to the OpenAI API and retrieves the response.
func callChatGPT(conversation []Message, cfg *utils.Config) (string, error) {
	requestBody, err := json.Marshal(createChatGPTRequest(conversation))
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var chatResponse ChatResponse
	err = json.Unmarshal(body, &chatResponse)
	if err != nil {
		return "", err
	}

	if len(chatResponse.Choices) > 0 {
		return chatResponse.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf(string(body))
}
