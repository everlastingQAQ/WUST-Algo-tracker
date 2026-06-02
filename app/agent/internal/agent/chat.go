package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"cwxu-algo/app/agent/internal/agent/tool"
	"cwxu-algo/app/common/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type Chat struct {
	conf   *conf.Agent
	client *http.Client
	apiKey string
	model  string
	url    string
}

func NewChat(cfg *conf.Agent) *Chat {
	configuredModel := ""
	configuredSecret := ""
	if cfg != nil {
		configuredModel = cfg.Model
		configuredSecret = cfg.Secret
	}
	modelName := firstNonEmpty(os.Getenv("AI_MODEL"), os.Getenv("DEEPSEEK_MODEL"), configuredModel)
	apiKey := firstNonEmpty(os.Getenv("AI_API_KEY"), os.Getenv("DEEPSEEK_API_KEY"), configuredSecret)
	baseURL := firstNonEmpty(os.Getenv("AI_BASE_URL"), os.Getenv("DEEPSEEK_BASE_URL"), "https://api.deepseek.com")
	return &Chat{
		conf:   cfg,
		client: &http.Client{Timeout: 120 * time.Second},
		apiKey: apiKey,
		model:  modelName,
		url:    chatCompletionsURL(baseURL),
	}
}

func (c *Chat) Chat(messages []*model.ChatCompletionMessage, tools ...tool.AgentToolFactory) (string, error) {
	ctx := context.Background()
	finalResp := ""
	reg := map[string]tool.AgentToolFactory{}
	toolUse := make([]compatTool, 0)
	for _, t := range tools {
		desc := t.Description()
		if desc == nil || desc.Function == nil {
			continue
		}
		reg[desc.Function.Name] = t
		toolUse = append(toolUse, compatTool{
			Type: fmt.Sprint(desc.Type),
			Function: compatFunction{
				Name:        desc.Function.Name,
				Description: desc.Function.Description,
				Parameters:  desc.Function.Parameters,
			},
		})
	}
	compatMessages := toCompatMessages(messages)
	for {
		req := compatChatRequest{
			Model:    c.model,
			Messages: compatMessages,
			Tools:    toolUse,
		}
		resp, err := c.createChatCompletion(ctx, &req)
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", errors.New("模型返回空")
		}
		msg := resp.Choices[0].Message
		if msg.Content != "" {
			finalResp = msg.Content
		}
		if msg.ReasoningContent != "" {
			fmt.Println(msg.ReasoningContent)
		}
		if resp.Choices[0].FinishReason != "tool_calls" || len(msg.ToolCalls) == 0 {
			break
		}
		compatMessages = append(compatMessages, msg)
		for _, toolCall := range msg.ToolCalls {
			log.Infof("执行%s %s", toolCall.Function.Name, toolCall.Function.Arguments)
			fn, ok := reg[toolCall.Function.Name]
			if !ok {
				return "", fmt.Errorf("模型请求了未注册工具: %s", toolCall.Function.Name)
			}
			toolMsg := fn.AiInterface(toolCall.Function.Arguments)
			log.Infof("结果%s", toolMsg)
			compatMessages = append(compatMessages, compatMessage{
				Role:       "tool",
				Content:    toolMsg,
				ToolCallID: toolCall.ID,
			})
		}
	}
	return finalResp, nil
}

type compatChatRequest struct {
	Model    string          `json:"model"`
	Messages []compatMessage `json:"messages"`
	Tools    []compatTool    `json:"tools,omitempty"`
}

type compatMessage struct {
	Role             string           `json:"role"`
	Content          string           `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ToolCalls        []compatToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string           `json:"tool_call_id,omitempty"`
}

type compatTool struct {
	Type     string         `json:"type"`
	Function compatFunction `json:"function"`
}

type compatFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type compatToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function compatToolFunction `json:"function"`
}

type compatToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type compatChatResponse struct {
	Choices []struct {
		FinishReason string        `json:"finish_reason"`
		Message      compatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (c *Chat) createChatCompletion(ctx context.Context, req *compatChatRequest) (*compatChatResponse, error) {
	if c.apiKey == "" {
		return nil, errors.New("AI_API_KEY/DEEPSEEK_API_KEY 未配置")
	}
	if req.Model == "" {
		return nil, errors.New("AI_MODEL/DEEPSEEK_MODEL 未配置")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	var resp compatChatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析模型响应失败: %w, body=%s", err, string(respBody))
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		if resp.Error != nil && resp.Error.Message != "" {
			return nil, fmt.Errorf("模型请求失败: status=%d type=%s message=%s", httpResp.StatusCode, resp.Error.Type, resp.Error.Message)
		}
		return nil, fmt.Errorf("模型请求失败: status=%d body=%s", httpResp.StatusCode, string(respBody))
	}
	return &resp, nil
}

func toCompatMessages(messages []*model.ChatCompletionMessage) []compatMessage {
	out := make([]compatMessage, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		content := ""
		if msg.Content != nil && msg.Content.StringValue != nil {
			content = *msg.Content.StringValue
		}
		out = append(out, compatMessage{
			Role:    fmt.Sprint(msg.Role),
			Content: content,
		})
	}
	return out
}

func chatCompletionsURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
