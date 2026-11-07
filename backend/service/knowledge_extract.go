package service

import (
	"archive/zip"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	pdf "github.com/dslipak/pdf"

	"github.com/nodal/nodal-server/apperr"
)

// 支持的文件类型：纯文本/代码类直接读，PDF 抽文字，DOCX 解 document.xml。
var knowledgeExtMIME = map[string]string{
	".txt": "text/plain", ".md": "text/markdown", ".markdown": "text/markdown",
	".csv": "text/csv", ".json": "application/json", ".log": "text/plain",
	".yaml": "text/yaml", ".yml": "text/yaml", ".html": "text/html", ".htm": "text/html",
	".xml": "application/xml",
	".go":  "text/x-go", ".py": "text/x-python", ".js": "text/javascript",
	".ts": "text/typescript", ".tsx": "text/typescript", ".jsx": "text/javascript",
	".java": "text/x-java", ".rs": "text/x-rust", ".c": "text/x-c",
	".h": "text/x-c", ".cpp": "text/x-c++", ".sql": "text/x-sql", ".sh": "text/x-sh",
	".pdf": "application/pdf", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
}

const maxChunksPerFile = 500

func normalizeExt(name string) string {
	return strings.ToLower(filepath.Ext(strings.TrimSpace(name)))
}

func supportedExt(ext string) (string, bool) {
	mime, ok := knowledgeExtMIME[ext]
	return mime, ok
}

func extractText(path, ext string) (string, error) {
	switch ext {
	case ".pdf":
		return extractPDF(path)
	case ".docx":
		return extractDOCX(path)
	case ".html", ".htm":
		text, err := readPlain(path)
		if err != nil {
			return "", err
		}
		return stripHTML(text), nil
	default:
		if _, ok := supportedExt(ext); !ok {
			return "", apperr.Newf(apperr.CodeUnsupportedFile, "不支持的文件类型 %s", ext)
		}
		return readPlain(path)
	}
}

func readPlain(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileParseFailed, "读取文件失败", err)
	}
	if !utf8.Valid(raw) {
		return "", apperr.New(apperr.CodeFileParseFailed, "文件不是 UTF-8 文本，无法解析")
	}
	return string(raw), nil
}

func extractPDF(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileParseFailed, "PDF 打开失败", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileParseFailed, "PDF 读取失败", err)
	}
	reader, err := pdf.NewReader(f, info.Size())
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileParseFailed, "PDF 解析失败", err)
	}

	plain, err := reader.GetPlainText()
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileParseFailed, "PDF 文本抽取失败", err)
	}
	raw, err := io.ReadAll(plain)
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileParseFailed, "PDF 文本读取失败", err)
	}
	return string(raw), nil
}

var docxParagraphEnd = regexp.MustCompile(`</w:p>|</w:tr>`)
var xmlTag = regexp.MustCompile(`<[^>]+>`)

func extractDOCX(path string) (string, error) {
	zipReader, err := zip.OpenReader(path)
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileParseFailed, "DOCX 不是合法压缩包", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", apperr.Wrap(apperr.CodeFileParseFailed, "DOCX 读取失败", err)
		}
		defer rc.Close()

		raw, err := io.ReadAll(rc)
		if err != nil {
			return "", apperr.Wrap(apperr.CodeFileParseFailed, "DOCX 内容读取失败", err)
		}
		text := string(raw)
		text = docxParagraphEnd.ReplaceAllString(text, "\n")
		text = xmlTag.ReplaceAllString(text, "")
		return html.UnescapeString(text), nil
	}
	return "", apperr.New(apperr.CodeFileParseFailed, "DOCX 中找不到 word/document.xml")
}

var multiNewline = regexp.MustCompile(`\n{3,}`)

func stripHTML(s string) string {
	s = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</\1>`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>|</li>`).ReplaceAllString(s, "\n")
	s = xmlTag.ReplaceAllString(s, " ")
	return html.UnescapeString(s)
}

// cleanKnowledgeText 统一换行、压缩空白；保留段落结构给切片用。
func cleanKnowledgeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\x00", "")
	s = multiNewline.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// splitKnowledgeChunks 按字符窗口切片，尽量在句子/段落边界切开，并保留 overlap 重叠。
func splitKnowledgeChunks(text string, size, overlap int) ([]string, bool) {
	if size <= 0 {
		size = 800
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}

	runes := []rune(text)
	chunks := make([]string, 0, len(runes)/size+1)
	truncated := false

	for start := 0; start < len(runes); {
		if len(chunks) >= maxChunksPerFile {
			truncated = true
			break
		}

		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		if end < len(runes) {
			// 在窗口最后 25% 内找一个句末/换行边界，避免把句子拦腰截断
			lower := start + size*3/4
			for i := end; i > lower; i-- {
				if isChunkBoundary(runes[i-1]) {
					end = i
					break
				}
			}
		}

		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end >= len(runes) {
			break
		}
		next := end - overlap
		if next <= start {
			next = start + 1
		}
		start = next
	}
	return chunks, truncated
}

func isChunkBoundary(r rune) bool {
	switch r {
	case '\n', '。', '！', '？', '；', '.', '!', '?', ';':
		return true
	default:
		return false
	}
}

// sanitizeFileName 只用于展示，落盘用 uuid。
func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." {
		name = "未命名文件"
	}
	if utf8.RuneCountInString(name) > 120 {
		runes := []rune(name)
		name = string(runes[:120])
	}
	return name
}
