// Copyright 2025 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}

// ThoughtItem is a thought that the tool appends to the log items.
type ThoughtItem struct {
	Thought   string `json:"thought"`
	CreatedAt string `json:"created_at"`
}

// ThinkTool is a tool that allows to think about something. It appends a thought to the log items.
type ThinkTool struct {
	mu       sync.Mutex
	thoughts []ThoughtItem // A lot of thoughts are needed to solve a problem
}

type ThinkInput struct {
	Thought string `json:"thought" jsonschema:"a thought to record"`
}

// Think is a tool that allows to think about something. It appends a thought to the log items.
func (t *ThinkTool) Think(ctx context.Context, sess *mcp.ServerSession, params *mcp.CallToolParamsFor[ThinkInput]) (*mcp.CallToolResultFor[any], error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	thought := params.Arguments.Thought
	if len(thought) == 0 {
		return nil, errors.New("no thoughts provided")
	}

	t.thoughts = append(t.thoughts, ThoughtItem{
		Thought:   thought,
		CreatedAt: time.Now().Format(time.RFC3339),
	})
	return &mcp.CallToolResultFor[any]{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Thought: %s", tidyThought(thought))}}}, nil
}

// GetThoughts is a tool that returns the thoughts recorded so far.
func (t *ThinkTool) GetThoughts(ctx context.Context, sess *mcp.ServerSession, params *mcp.CallToolParamsFor[struct{}]) (*mcp.CallToolResultFor[any], error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.thoughts) == 0 {
		return nil, errors.New("no thoughts recorded. Use the think tool to record a thought first.")
	}

	thoughts := []string{}
	for i, thought := range t.thoughts {
		thoughts = append(thoughts, fmt.Sprintf("Thought #%d at %s:\n%s\n", i+1, thought.CreatedAt, thought.Thought))
	}
	return &mcp.CallToolResultFor[any]{Content: []mcp.Content{&mcp.TextContent{Text: strings.Join(thoughts, "\n")}}}, nil
}

func (t *ThinkTool) ClearThoughts(ctx context.Context, sess *mcp.ServerSession, params *mcp.CallToolParamsFor[struct{}]) (*mcp.CallToolResultFor[any], error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.thoughts = []ThoughtItem{}
	return &mcp.CallToolResultFor[any]{Content: []mcp.Content{&mcp.TextContent{Text: "Thoughts cleared."}}}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "think-tool",
		Version: "v0.0.1",
	}, nil)

	thinkTool := &ThinkTool{}

	mcp.AddTool(server, &mcp.Tool{
		Name: "think",
		Description: `Use this tool to think about something.
It will not obtain new information or change anything, but just append the thought to the log.
Use it when complex reasoning or cache memory is needed.`,
	}, thinkTool.Think)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_thoughts",
		Description: `Retrieve all thoughts recorded in the current session. This tool helps review the thinking process that has occurred so far.`,
	}, thinkTool.GetThoughts)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "clear_thoughts",
		Description: `Clear all recorded thoughts from the current session. Use this to start fresh if the thinking process needs to be reset.`,
	}, thinkTool.ClearThoughts)

	logger := slog.Default()
	logger.Info("starting mcp stdio server ...")
	if err := server.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		logger.Error("failed to run server", slog.Any("error", err))
	}
}

func tidyThought(thought string) string {
	if len(thought) > 50 {
		return thought[:50] + "..."
	}
	return thought
}
