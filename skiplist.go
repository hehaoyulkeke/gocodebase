package gocodebase

import (
	"math"
	"math/rand"
	"sync"
)

const (
	// DefaultMaxLevel 默认skip list最大深度
	DefaultMaxLevel int = 18
	// DefaultProbability 默认的概率
	DefaultProbability float64 = 1 / math.E
)

// elementNode 数组指针，指向元素
type elementNode struct {
	next []*Element
}

// Element 跳转表数据结构
type Element struct {
	elementNode
	key   float64
	value interface{} // 定义元素
}

// Key 获取key的值
func (e *Element) Key() float64 {
	return e.key
}

// Value 获取key的值
func (e *Element) Value() interface{} {
	return e.value
}


type SkipList struct {
	elementNode
	maxLevel       int            // 最大深度
	length         int            // 长度
	randSource     rand.Source    // 动态调节跳转表的长度
	probability    float64        // 概率
	probTable      []float64      // 存储位置，对应key
	mutex          sync.RWMutex   // 保证线程安全
	prevNodesCache []*elementNode // 缓存
}

// NewSkipList 新建跳转表
func NewSkipList() *SkipList {
	return NewWithMaxLevel(DefaultMaxLevel)
}

// ProbabilityTable 初始化 Probability Table
func ProbabilityTable(probability float64, maxLevel int) (table []float64) {
	for i := 1; i <= maxLevel; i++ {
		prob := math.Pow(probability, float64(i-1))
		table = append(table, prob)
	}
	return table
}

// NewWithMaxLevel 自定义maxLevel新建跳转表
func NewWithMaxLevel(maxLevel int) *SkipList {
	if maxLevel < 1 || maxLevel > DefaultMaxLevel {
		panic("invalid maxlevel")
	}

	return &SkipList{
		elementNode:    elementNode{next: make([]*Element, maxLevel)},
		prevNodesCache: make([]*elementNode, maxLevel),
		maxLevel:       maxLevel,
		randSource:     rand.New(rand.NewSource(42)),
		probability:    DefaultProbability,
		probTable:      ProbabilityTable(DefaultProbability, maxLevel),
	}
}

// 随机计算最接近的
func (list *SkipList) randLevel() (level int) {
	r := float64(list.randSource.Int63()) / (1 << 63)
	level = 1
	for level < list.maxLevel && r < list.probTable[level] {
		level++ // 级别追加
	}

	return level
}

// SetProbability 设置新的概率,刷新概率表
func (list *SkipList) SetProbability(newProbability float64) {
	list.probability = newProbability
	list.probTable = ProbabilityTable(newProbability, list.maxLevel)
}

// Set 存储新的值
func (list *SkipList) Set(key float64, value interface{}) *Element {
	list.mutex.Lock()
	defer list.mutex.Unlock() // 线程安全

	var element *Element
	prevs := list.getPrevElementNodes(key)
	if element = prevs[0].next[0]; element != nil && key == element.key {
		element.value = value
		return element
	}

	element = &Element{
		elementNode: elementNode{next: make([]*Element, list.randLevel())},
		key:         key,
		value:       value,
	}
	list.length++

	for i := range element.next { // 插入数据
		element.next[i] = prevs[i].next[i]
		prevs[i].next[i] = element // 记录位置
	}

	return element
}

// Get 获取key对应的值
func (list *SkipList) Get(key float64) *Element {
	list.mutex.Lock()
	defer list.mutex.Unlock() // 线程安全

	var prev *elementNode = &list.elementNode // 保存前置结点
	var next *Element

	for i := list.maxLevel - 1; i >= 0; i-- {
		next = prev.next[i] // 循环跳到下一个
		for next != nil && key > next.key {
			prev = &next.elementNode
			next = next.next[i]
		}
	}

	if next != nil && next.key == key { // 找到
		return next
	}

	return nil // 没有找到
}

// Remove 获取key对应的值
func (list *SkipList) Remove(key float64) *Element {
	list.mutex.Lock()
	defer list.mutex.Unlock() // 线程安全

	var element *Element
	prevs := list.getPrevElementNodes(key)
	if element = prevs[0].next[0]; element != nil && key == element.key {
		for k, v := range element.next {
			prevs[k].next[k] = v // 删除
		}

		list.length--
		return element
	}

	return nil
}

func (list *SkipList) getPrevElementNodes(key float64) []*elementNode {
	var prev *elementNode = &list.elementNode // 保存前置结点
	var next *Element
	prevs := list.prevNodesCache // 缓冲集合
	for i := list.maxLevel - 1; i >= 0; i-- {
		next = prev.next[i] // 循环跳到下一个
		for next != nil && key > next.key {
			prev = &next.elementNode
			next = next.next[i]
		}
		prevs[i] = prev
	}
	return prevs
}
