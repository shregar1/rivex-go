package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// Slugify converts text to a URL-friendly slug.
//
// It lowercases the input, strips accents via Unicode NFKD decomposition,
// replaces non-alphanumeric characters with the separator, collapses
// consecutive separators, and trims leading/trailing separators.
//
// Example:
//
//	rivex.Slugify("Hello World!", "-")  // "hello-world"
//	rivex.Slugify("Cafe Latte", "_")   // "cafe_latte"
func Slugify(text, separator string) string {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	cSep := C.CString(separator)
	defer C.free(unsafe.Pointer(cSep))

	result := C.rivex_slugify(cText, cSep)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// EscapeHTML escapes HTML special characters in the input text.
//
// The following characters are replaced with their HTML entity equivalents:
// & -> &amp;  < -> &lt;  > -> &gt;  " -> &quot;  ' -> &#x27;
func EscapeHTML(text string) string {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	result := C.rivex_escape_html(cText)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// MarkdownToHTML converts a Markdown string to HTML.
func MarkdownToHTML(md string) string {
	cMD := C.CString(md)
	defer C.free(unsafe.Pointer(cMD))
	result := C.rivex_markdown_to_html(cMD)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// LevenshteinDistance computes the Levenshtein edit distance between two strings.
func LevenshteinDistance(a, b string) int {
	cA := C.CString(a)
	defer C.free(unsafe.Pointer(cA))
	cB := C.CString(b)
	defer C.free(unsafe.Pointer(cB))
	return int(C.rivex_levenshtein_distance(cA, cB))
}

// StripHTMLTags removes all HTML tags from the input, returning plain text.
func StripHTMLTags(html string) string {
	cHTML := C.CString(html)
	defer C.free(unsafe.Pointer(cHTML))
	result := C.rivex_strip_html_tags(cHTML)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// RenderTemplate renders a Mustache-style template with the given JSON variables.
func RenderTemplate(tmpl, varsJSON string) string {
	cTmpl := C.CString(tmpl)
	defer C.free(unsafe.Pointer(cTmpl))
	cVars := C.CString(varsJSON)
	defer C.free(unsafe.Pointer(cVars))
	result := C.rivex_render_template(cTmpl, cVars)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// TextDiff computes a unified text diff between two strings.
func TextDiff(a, b string) string {
	cA := C.CString(a)
	defer C.free(unsafe.Pointer(cA))
	cB := C.CString(b)
	defer C.free(unsafe.Pointer(cB))
	result := C.rivex_text_diff(cA, cB)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// SimilarityScore computes a similarity score (0.0 to 1.0) between two strings.
func SimilarityScore(a, b string) float64 {
	cA := C.CString(a)
	defer C.free(unsafe.Pointer(cA))
	cB := C.CString(b)
	defer C.free(unsafe.Pointer(cB))
	return float64(C.rivex_similarity_score(cA, cB))
}

// FuzzyMatch returns true if the similarity between pattern and text meets or exceeds the threshold.
func FuzzyMatch(pattern, text string, threshold float64) bool {
	cPattern := C.CString(pattern)
	defer C.free(unsafe.Pointer(cPattern))
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	return C.rivex_fuzzy_match(cPattern, cText, C.double(threshold)) == 1
}

// TokenizeWords tokenizes text into words and returns a JSON array string.
func TokenizeWords(text string) string {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	result := C.rivex_tokenize_words(cText)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// TokenizeSentences tokenizes text into sentences and returns a JSON array string.
func TokenizeSentences(text string) string {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	result := C.rivex_tokenize_sentences(cText)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}
