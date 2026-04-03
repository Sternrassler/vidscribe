package mcp

import (
	"os"
	"path/filepath"
)

// claudeCommands maps filename → content for Claude Code slash commands.
var claudeCommands = map[string]string{
	"vidscribe.md": `# vidscribe

Transkribiere ein Video mit dem vidscribe MCP-Tool (` + "`mcp__vidscribe__transcribe_video`" + `).

Argumente: $ARGUMENTS

Arbeite still — kein Kommentar während der Verarbeitung. Gib nur das Ergebnis aus oder eine kurze Fehlermeldung.

## Verhalten

- Wenn $ARGUMENTS eine URL enthält: direkt transkribieren
- Wenn $ARGUMENTS leer ist: kurz erklären, wie das Kommando zu verwenden ist

## Parameter-Defaults

- ` + "`cookies_browser`" + `: ` + "`chrome`" + ` (für YouTube Bot-Detection vermeiden)
- ` + "`language`" + `: automatisch erkennen (leer lassen)
- ` + "`model`" + `: Standard (kein Override nötig, CUDA wird automatisch genutzt)
- ` + "`output_dir`" + `: ` + "`./transcripts/`" + `

## Optionale Argumente (als Freitext erkannt)

Der Nutzer kann hinter der URL Hinweise geben, z.B.:

- ` + "`--lang de`" + ` oder ` + "`language=de`" + ` → ` + "`language: \"de\"`" + ` setzen
- ` + "`--model large`" + ` → ` + "`model: \"large\"`" + ` setzen
- ` + "`--no-cookies`" + ` → ` + "`cookies_browser`" + ` weglassen

## Ausgabe nach Transkription

Zeige:

1. Zieldatei(en) (` + "`.txt`" + ` / ` + "`.md`" + `)
2. Erkannte Sprache
3. Die ersten ~5 Sätze des Transkripts als Vorschau
`,

	"vidscribe-check.md": `# vidscribe-check

Prüfe ob alle vidscribe-Abhängigkeiten installiert sind.

Rufe ` + "`mcp__vidscribe__check_dependencies`" + ` auf und gib das Ergebnis aus.
Arbeite still — kein Kommentar, nur das Ergebnis.
`,

	"vidscribe-sites.md": `# vidscribe-sites

Zeige alle von vidscribe unterstützten Plattformen/Seiten.

Rufe ` + "`mcp__vidscribe__list_supported_sites`" + ` auf und gib die Liste aus.
Arbeite still — kein Kommentar, nur das Ergebnis.
`,
}

// installClaudeCommands writes vidscribe slash-command files to ~/.claude/commands/.
// Existing files are silently overwritten so they stay up to date.
// Errors are ignored: the MCP server should still start even if the home
// directory is unavailable or read-only.
func installClaudeCommands() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".claude", "commands")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	for name, content := range claudeCommands {
		_ = os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	}
}
