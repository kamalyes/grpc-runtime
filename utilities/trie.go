/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:00:00
 * @FilePath: \grpc-runtime\utilities\trie.go
 * @Description: Double Array trie 实现，用于路径前缀匹配
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package utilities

import (
	"sort"
)

// DoubleArray 基于 Double Array 的字符串序列 trie 实现
type DoubleArray struct {
	// Encoding 字符串到整数的编码映射
	Encoding map[string]int
	// Base Double Array 的 base 数组
	Base []int
	// Check Double Array 的 check 数组
	Check []int
}

// NewDoubleArray 从字符串序列集合构建 DoubleArray
func NewDoubleArray(seqs [][]string) *DoubleArray {
	da := &DoubleArray{Encoding: make(map[string]int)}
	if len(seqs) == 0 {
		return da
	}

	encoded := registerTokens(da, seqs)
	sort.Sort(byLex(encoded))

	root := node{row: -1, col: -1, left: 0, right: len(encoded)}
	addSeqs(da, encoded, 0, root)

	for i := len(da.Base); i > 0; i-- {
		if da.Check[i-1] != 0 {
			da.Base = da.Base[:i]
			da.Check = da.Check[:i]
			break
		}
	}
	return da
}

func registerTokens(da *DoubleArray, seqs [][]string) [][]int {
	var result [][]int
	for _, seq := range seqs {
		encoded := make([]int, 0, len(seq))
		for _, token := range seq {
			if _, ok := da.Encoding[token]; !ok {
				da.Encoding[token] = len(da.Encoding)
			}
			encoded = append(encoded, da.Encoding[token])
		}
		result = append(result, encoded)
	}
	for i := range result {
		result[i] = append(result[i], len(da.Encoding))
	}
	return result
}

type node struct {
	row, col    int
	left, right int
}

func (n node) value(seqs [][]int) int {
	return seqs[n.row][n.col]
}

func (n node) children(seqs [][]int) []*node {
	var result []*node
	lastVal := int(-1)
	last := new(node)
	for i := n.left; i < n.right; i++ {
		if lastVal == seqs[i][n.col+1] {
			continue
		}
		last.right = i
		last = &node{
			row:  i,
			col:  n.col + 1,
			left: i,
		}
		result = append(result, last)
	}
	last.right = n.right
	return result
}

func addSeqs(da *DoubleArray, seqs [][]int, pos int, n node) {
	ensureSize(da, pos)

	children := n.children(seqs)
	var i int
	for i = 1; ; i++ {
		ok := func() bool {
			for _, child := range children {
				code := child.value(seqs)
				j := i + code
				ensureSize(da, j)
				if da.Check[j] != 0 {
					return false
				}
			}
			return true
		}()
		if ok {
			break
		}
	}
	da.Base[pos] = i
	for _, child := range children {
		code := child.value(seqs)
		j := i + code
		da.Check[j] = pos + 1
	}
	terminator := len(da.Encoding)
	for _, child := range children {
		code := child.value(seqs)
		if code == terminator {
			continue
		}
		j := i + code
		addSeqs(da, seqs, j, *child)
	}
}

func ensureSize(da *DoubleArray, i int) {
	for i >= len(da.Base) {
		da.Base = append(da.Base, make([]int, len(da.Base)+1)...)
		da.Check = append(da.Check, make([]int, len(da.Check)+1)...)
	}
}

type byLex [][]int

func (l byLex) Len() int      { return len(l) }
func (l byLex) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l byLex) Less(i, j int) bool {
	si := l[i]
	sj := l[j]
	var k int
	for k = 0; k < len(si) && k < len(sj); k++ {
		if si[k] < sj[k] {
			return true
		}
		if si[k] > sj[k] {
			return false
		}
	}
	return k < len(sj)
}

// HasCommonPrefix 判断 DoubleArray 中是否有序列是给定序列的前缀
func (da *DoubleArray) HasCommonPrefix(seq []string) bool {
	if len(da.Base) == 0 {
		return false
	}

	var i int
	for _, t := range seq {
		code, ok := da.Encoding[t]
		if !ok {
			break
		}
		j := da.Base[i] + code
		if len(da.Check) <= j || da.Check[j] != i+1 {
			break
		}
		i = j
	}
	j := da.Base[i] + len(da.Encoding)
	if len(da.Check) <= j || da.Check[j] != i+1 {
		return false
	}
	return true
}
