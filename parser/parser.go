package parser

import (
	"regexp"
	"strings"
)

// LogRecord 代表解析后的结构化日志
type LogRecord struct {
	Time       string `json:"time"`
	SrcIP      string `json:"src_ip"`
	SrcPort    string `json:"src_port"`
	Protocol   string `json:"protocol"` // tcp 或 udp
	DstIP      string `json:"dst_ip"`
	DstPort    string `json:"dst_port"`
	Inbound    string `json:"inbound"`
	Outbound   string `json:"outbound"`
	Email      string `json:"email"`
}

// XrayRegex 完美的匹配正则
// 考虑到日志中包含了 "from 106..." 和 "from tcp:106..." 两种格式
var XrayRegex = regexp.MustCompile(`^(?P<time>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) (?:from )?(?:(?P<proto_from>tcp|udp):)?(?P<src_ip>[\d\.]+):(?P<src_port>\d+) accepted (?P<proto>\w+):(?P<dst_ip>[^:]+):(?P<dst_port>\d+) \[(?P<inbound>\w+) -> (?P<outbound>\w+)\] email: (?P<email>.+)$`)

// ParseLine 解析单行日志
func ParseLine(line string) (*LogRecord, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false
	}

	matches := XrayRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil, false // 无法匹配的非法行或空行直接跳过
	}

	// 映射正则捕获组
	result := make(map[string]string)
	for i, name := range XrayRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = matches[i]
		}
	}

	return &LogRecord{
		Time:     result["time"],
		SrcIP:    result["src_ip"],
		SrcPort:  result["src_port"],
		Protocol: result["proto"],
		DstIP:    result["dst_ip"],
		DstPort:  result["dst_port"],
		Inbound:  result["inbound"],
		Outbound: result["outbound"],
		Email:    result["email"],
	}, true
}