package skillz

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const serverName = "Skillz MCP Server"
const serverVersion = "0.1.0-go"

type RunOptions struct {
	Transport string
	Host      string
	Port      int
	Path      string
}

func BuildMCPServer(registry *Registry) *server.MCPServer {
	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithInstructions(buildServerInstructions(registry)),
		server.WithResourceCapabilities(true, false),
		server.WithToolCapabilities(true),
	)

	registerFetchResourceTool(mcpServer, registry)
	for _, skill := range registry.Skills() {
		resourceMetadata := registerSkillResources(mcpServer, skill)
		registerSkillTool(mcpServer, skill, resourceMetadata)
	}

	return mcpServer
}

func RunMCPServer(ctx context.Context, mcpServer *server.MCPServer, options RunOptions) error {
	transport := strings.ToLower(strings.TrimSpace(options.Transport))
	if transport == "" {
		transport = "stdio"
	}

	switch transport {
	case "stdio":
		stdioServer := server.NewStdioServer(mcpServer)
		return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
	case "http":
		address := fmt.Sprintf("%s:%d", options.Host, options.Port)
		streamingServer := server.NewStreamableHTTPServer(
			mcpServer,
			server.WithEndpointPath(options.Path),
		)
		return streamingServer.Start(address)
	case "sse":
		address := fmt.Sprintf("%s:%d", options.Host, options.Port)
		sseServer := server.NewSSEServer(
			mcpServer,
			server.WithBasePath(options.Path),
		)
		return sseServer.Start(address)
	default:
		return fmt.Errorf("unsupported transport: %s", options.Transport)
	}
}

func registerFetchResourceTool(mcpServer *server.MCPServer, registry *Registry) {
	fetchTool := mcp.NewTool(
		"fetch_resource",
		mcp.WithDescription(
			"[FALLBACK ONLY] Fetch a skill resource by URI. " +
				"Use this only when native MCP resource fetching is unavailable.",
		),
		mcp.WithString("resource_uri", mcp.Description("resource://skillz/{skill-slug}/{path}"), mcp.Required()),
	)

	mcpServer.AddTool(fetchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceURI := request.GetString("resource_uri", "")
		if resourceURI == "" {
			result := makeErrorResource("(missing)", "resource_uri is required")
			return mcp.NewToolResultStructured(result, "resource_uri is required"), nil
		}

		result := FetchResourceJSON(registry, resourceURI)
		return mcp.NewToolResultStructured(result, "resource fetched"), nil
	})
}

func registerSkillResources(mcpServer *server.MCPServer, skill Skill) []ResourceMetadata {
	metadata := make([]ResourceMetadata, 0, len(skill.Resources))

	for _, relPath := range sortedKeys(skill.Resources) {
		boundRelPath := relPath
		uri := BuildResourceURI(skill, boundRelPath)
		name := skill.ResourceName(boundRelPath)
		mimeType := detectMimeType(boundRelPath)

		resource := mcp.NewResource(
			uri,
			name,
			mcp.WithMIMEType(toOptionalString(mimeType)),
		)

		mcpServer.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			_ = request
			data, err := skill.OpenBytes(boundRelPath)
			if err != nil {
				return nil, err
			}
			if utf8Bytes(data) {
				content := mcp.TextResourceContents{
					URI:      uri,
					MIMEType: toOptionalString(mimeType),
					Text:     string(data),
				}
				return []mcp.ResourceContents{content}, nil
			}

			content := mcp.BlobResourceContents{
				URI:      uri,
				MIMEType: toOptionalString(mimeType),
				Blob:     base64.StdEncoding.EncodeToString(data),
			}
			return []mcp.ResourceContents{content}, nil
		})

		metadata = append(metadata, ResourceMetadata{
			URI:      uri,
			Name:     name,
			MIMEType: mimeType,
		})
	}

	return metadata
}

func registerSkillTool(mcpServer *server.MCPServer, skill Skill, resources []ResourceMetadata) {
	tool := mcp.NewTool(
		skill.Slug,
		mcp.WithDescription(formatSkillDescription(skill)),
		mcp.WithString("task", mcp.Description("The user task for this skill"), mcp.Required()),
	)

	mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		task := strings.TrimSpace(request.GetString("task", ""))
		if task == "" {
			return nil, errors.New("the 'task' parameter must be a non-empty string")
		}

		response := map[string]any{
			"skill": taskSkillSlug(skill),
			"task":  task,
			"metadata": map[string]any{
				"name":          skill.Metadata.Name,
				"description":   skill.Metadata.Description,
				"license":       skill.Metadata.License,
				"allowed_tools": skill.Metadata.AllowedTools,
				"extra":         skill.Metadata.Extra,
			},
			"resources":    resources,
			"instructions": skill.Instructions,
			"usage":        defaultUsageText(),
		}

		return mcp.NewToolResultStructured(response, "skill instructions returned"), nil
	})
}

func buildServerInstructions(registry *Registry) string {
	names := make([]string, 0, len(registry.Skills()))
	for _, skill := range registry.Skills() {
		names = append(names, skill.Metadata.Name)
	}
	if len(names) == 0 {
		return "Skillz MCP server exposing local skills as tools and resources."
	}
	return fmt.Sprintf(
		"Skillz MCP server exposing %d skill(s): %s",
		len(names),
		strings.Join(names, ", "),
	)
}

func formatSkillDescription(skill Skill) string {
	description := strings.TrimSpace(skill.Metadata.Description)
	if description == "" {
		description = "Skill instructions"
	}
	return "[SKILL] " + description + " - Invoke this tool to receive specialized instructions and resources for the task."
}

func defaultUsageText() string {
	return "Read the skill instructions, retrieve needed resources via MCP resources (or fetch_resource fallback), then apply the guidance to complete the task."
}

func taskSkillSlug(skill Skill) string {
	return skill.Slug
}

func toOptionalString(value any) string {
	if value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func utf8Bytes(data []byte) bool {
	return utf8.Valid(data)
}
