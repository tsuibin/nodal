package embed

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// mockEmbedder 生成确定性「词袋哈希」向量：
// 把文本切成中文字符 bigram / 英文单词，逐个哈希到固定维度并累加。
// 共享词汇越多、余弦相似度越高，因此不接真实模型也能演示 RAG 召回。
type mockEmbedder struct {
	dim int
}

func (m *mockEmbedder) ID() string        { return "mock/embed" }
func (m *mockEmbedder) Provider() string  { return "mock" }
func (m *mockEmbedder) RequiresKey() bool { return false }

func (m *mockEmbedder) Embed(_ context.Context, _ Credential, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		out = append(out, mockVector(text, m.dim))
	}
	return out, nil
}

func mockVector(text string, dim int) []float32 {
	vec := make([]float32, dim)
	text = strings.ToLower(text)

	// 中文按字符 bigram，英文/数字按词
	var tokens []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			tokens = append(tokens, word.String())
			word.Reset()
		}
	}

	var prevRune rune
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) && r < 128, unicode.IsDigit(r):
			flush()
			v := vec[hashIndex(string(r), dim)]
			vec[hashIndex(string(r), dim)] = v + 1
		case unicode.Is(unicode.Han, r):
			flush()
			if prevRune != 0 && unicode.Is(unicode.Han, prevRune) {
				tokens = append(tokens, string([]rune{prevRune, r}))
			}
			prevRune = r
			continue
		default:
			flush()
		}
		prevRune = r
	}
	flush()

	if len(tokens) == 0 {
		// 兜底：整段文本哈希一次，避免全零向量无法计算余弦
		vec[hashIndex(text, dim)] = 1
	}
	for _, token := range tokens {
		idx := hashIndex(token, dim)
		vec[idx] += 1
	}

	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		n := float32(math.Sqrt(norm))
		for i := range vec {
			vec[i] /= n
		}
	}
	return vec
}

func hashIndex(token string, dim int) int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(token))
	return int(h.Sum64() % uint64(dim))
}
