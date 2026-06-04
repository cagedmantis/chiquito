package spell

// builtinWords is a small fallback vocabulary used when no system word list is
// available. It is intentionally modest — enough to be useful for prose and to
// recognize chiquito's own terms — not a complete dictionary.
var builtinWords = []string{
	// Common English words.
	"a", "an", "and", "the", "this", "that", "these", "those", "is", "are",
	"was", "were", "be", "been", "being", "am", "do", "does", "did", "have",
	"has", "had", "will", "would", "shall", "should", "can", "could", "may",
	"might", "must", "of", "to", "in", "on", "at", "by", "for", "with", "from",
	"as", "into", "over", "under", "about", "above", "below", "between", "if",
	"then", "else", "when", "while", "because", "so", "but", "or", "not", "no",
	"yes", "all", "any", "some", "each", "every", "few", "more", "most", "other",
	"such", "only", "own", "same", "than", "too", "very", "just", "also",
	"hello", "world", "text", "editor", "file", "files", "line", "lines",
	"word", "words", "code", "source", "search", "replace", "save", "open",
	"quick", "brown", "fox", "jumps", "lazy", "dog", "test", "testing", "spell",
	"check", "checker", "cursor", "buffer", "config", "default", "color",
	"theme", "syntax", "terminal", "keyboard", "command", "comment", "string",
	"number", "function", "package", "import", "return", "error", "value",
	// chiquito-specific vocabulary.
	"chiquito", "emacs", "markdown", "unicode", "toml",
}
