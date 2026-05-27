/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\metadata\timeout.go
 * @Description: gRPC 超时解码，将 grpc-timeout header 解析为 time.Duration
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package metadata

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// 超时单位映射
var timeoutUnitMap = map[byte]time.Duration{
	'H': time.Hour,
	'M': time.Minute,
	'S': time.Second,
	'm': time.Millisecond,
	'u': time.Microsecond,
	'n': time.Nanosecond,
}

// DecodeTimeout 将 gRPC 超时字符串解码为 time.Duration
// 格式为 <数值><单位>，如 "1H"、"500m"、"100u"
func DecodeTimeout(timeout string) (time.Duration, error) {
	if len(timeout) < 2 {
		return 0, errors.New("timeout string too short")
	}
	unit := timeout[len(timeout)-1]
	d, ok := timeoutUnitMap[unit]
	if !ok {
		return 0, fmt.Errorf("timeout unit %q not recognized", unit)
	}
	num, err := strconv.ParseInt(timeout[:len(timeout)-1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("timeout value %q not valid: %w", timeout[:len(timeout)-1], err)
	}
	return time.Duration(num) * d, nil
}

// EncodeTimeout 将 time.Duration 编码为 gRPC 超时字符串
func EncodeTimeout(d time.Duration) string {
	switch {
	case d%time.Hour == 0:
		return fmt.Sprintf("%dH", d/time.Hour)
	case d%time.Minute == 0:
		return fmt.Sprintf("%dM", d/time.Minute)
	case d%time.Second == 0:
		return fmt.Sprintf("%dS", d/time.Second)
	case d%time.Millisecond == 0:
		return fmt.Sprintf("%dm", d/time.Millisecond)
	case d%time.Microsecond == 0:
		return fmt.Sprintf("%du", d/time.Microsecond)
	default:
		return fmt.Sprintf("%dn", d/time.Nanosecond)
	}
}
