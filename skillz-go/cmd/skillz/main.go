package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/intellectronica/skillz/skillz-go/internal/skillz"
)

func main() {
	home, _ := os.UserHomeDir()
	defaultRoot := filepath.Join(home, ".skillz")

	listSkills := flag.Bool("list-skills", false, "List parsed skills and exit")
	fetchResource := flag.String("fetch-resource", "", "Fetch a resource by URI and print JSON")
	transport := flag.String("transport", "stdio", "Transport: stdio, http, sse")
	host := flag.String("host", "127.0.0.1", "Host for HTTP/SSE transport")
	port := flag.Int("port", 8000, "Port for HTTP/SSE transport")
	path := flag.String("path", "/mcp", "Path for HTTP/SSE transport")
	flag.Parse()

	skillsRoot := defaultRoot
	args := flag.Args()
	if len(args) > 0 && args[0] != "" {
		skillsRoot = args[0]
	}

	registry := skillz.NewRegistry(skillsRoot)
	if err := registry.Load(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *listSkills {
		skills := registry.Skills()
		if len(skills) == 0 {
			fmt.Println("No valid skills discovered.")
			return
		}
		for _, item := range skills {
			fmt.Printf("- %s (slug: %s) -> %s\n", item.Metadata.Name, item.Slug, item.Directory)
		}
		return
	}

	if *fetchResource != "" {
		result := skillz.FetchResourceJSON(registry, *fetchResource)
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal result: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(encoded))
		return
	}

	mcpServer := skillz.BuildMCPServer(registry)
	runOptions := skillz.RunOptions{
		Transport: *transport,
		Host:      *host,
		Port:      *port,
		Path:      *path,
	}

	if err := skillz.RunMCPServer(context.Background(), mcpServer, runOptions); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
