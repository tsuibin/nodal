package llm

import (
	"bufio"
	"io"
	"strings"
)

// scanSSE 解析标准 text/event-stream：按空行分帧，聚合 event / data 字段。
//
// 各厂商的 SSE 格式基本一致（OpenAI 兼容接口、Anthropic 都是），差异只在 JSON 结构，
// 因此把「读帧」抽出来共用，「解帧」留给各自适配器。
func scanSSE(r io.Reader, handle func(event, data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 32*1024), 1024*1024)

	var event string
	var data []string

	dispatch := func() error {
		if event == "" && len(data) == 0 {
			return nil
		}
		err := handle(event, strings.Join(data, "\n"))
		event = ""
		data = data[:0]
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if err := dispatch(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue // 注释/心跳
		}

		field, value, found := strings.Cut(line, ":")
		if found {
			value = strings.TrimPrefix(value, " ")
		}
		switch field {
		case "event":
			event = value
		case "data":
			data = append(data, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return dispatch()
}

// errStreamDone 是适配器内部用于提前结束流式读取的哨兵错误。
var errStreamDone = streamDoneError{}

type streamDoneError struct{}

func (streamDoneError) Error() string { return "llm: stream finished" }
