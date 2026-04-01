package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/devsternrassler/vidscribe/internal/deps"
	"github.com/devsternrassler/vidscribe/internal/mcp"
	"github.com/devsternrassler/vidscribe/internal/pipeline"
	"github.com/spf13/cobra"
)

var (
	model          string
	language       string
	outputDir      string
	cookiesBrowser string
	cookiesFile    string
	jsRuntime      string
	format         string
	engine         string
	mcpMode        bool
	verbose        bool
)

var rootCmd = &cobra.Command{
	Use:   "vidscribe [URL]",
	Short: "Transcribe audio from YouTube and 1000+ platforms",
	Long: `vidscribe downloads audio from any yt-dlp-supported platform and
transcribes it using faster-whisper (or openai-whisper as fallback).

Run 'vidscribe --mcp' to start in MCP server mode for Claude integration.`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
}

// SetVersion injects build-time version info from goreleaser ldflags.
func SetVersion(ver, commit, date string) {
	rootCmd.Version = ver + " (" + commit + ", " + date + ")"
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&model, "model", "small", "Whisper model: tiny|base|small|medium|large")
	rootCmd.Flags().StringVar(&language, "language", "auto", "Language (ISO 639-1) or 'auto'")
	rootCmd.Flags().StringVar(&outputDir, "output-dir", "./transcripts", "Output directory")
	rootCmd.Flags().StringVar(&cookiesBrowser, "cookies-browser", "", "Browser for cookie auth: chrome|firefox|safari|edge")
	rootCmd.Flags().StringVar(&cookiesFile, "cookies-file", "", "Path to Netscape cookie file")
	rootCmd.Flags().StringVar(&jsRuntime, "js-runtime", "", "JS runtime for yt-dlp extractor args: deno|node")
	rootCmd.Flags().StringVar(&format, "format", "txt,md", "Output formats: txt,md,json,srt,vtt (comma-separated)")
	rootCmd.Flags().StringVar(&engine, "engine", "faster", "Whisper engine: faster|openai")
	rootCmd.Flags().BoolVar(&mcpMode, "mcp", false, "Start as MCP server (stdio)")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose output")
}

func run(cmd *cobra.Command, args []string) error {
	if mcpMode {
		return mcp.Serve()
	}

	if len(args) == 0 {
		return fmt.Errorf("URL required\n\nUsage: vidscribe [URL] [flags]\nRun 'vidscribe --mcp' for MCP server mode")
	}

	url := args[0]

	if err := deps.Check(engine); err != nil {
		return err
	}

	cfg := &pipeline.Config{
		URL:            url,
		Model:          model,
		Language:       language,
		OutputDir:      outputDir,
		CookiesBrowser: cookiesBrowser,
		CookiesFile:    cookiesFile,
		JSRuntime:      jsRuntime,
		Formats:        strings.Split(format, ","),
		Engine:         engine,
		Verbose:        verbose,
	}

	paths, err := pipeline.Run(context.Background(), cfg, os.Stderr)
	if err != nil {
		return err
	}

	fmt.Println("Transcription complete. Files written:")
	for _, p := range paths {
		fmt.Println(" ", p)
	}
	return nil
}
