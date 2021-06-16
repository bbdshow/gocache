# gocache

[![Build Status](https://travis-ci.org/bbdshow/gocache.svg?branch=master)](https://travis-ci.org/bbdshow/gocache)
[![codecov](https://codecov.io/gh/bbdshow/gocache/branch/master/graph/badge.svg)](https://codecov.io/gh/bbdshow/gocache)
[![Go Report Card](https://goreportcard.com/badge/github.com/bbdshow/gocache)](https://goreportcard.com/report/github.com/bbdshow/gocache)

> 一款简易的内存缓存实现，支持容量控制，TTL和数据落盘。

## 特性
1. Cache 接口， 会采用 sync.Map 和 rwMutex + map 实现
2. 采用 sync.Map 实现， 在多核，大量读取，锁竞争多的情况下存在优势，缺点是内存占用高，空间换时间。 
3. 支持容量控制，到达限制后由多种策略控制内存使用量
3. 支持重启程序加载缓存内容，简单防止因重启导致的缓存击穿。
4. 支持 TTL Key 

## Usage

> 请查看 example 目录

## Notice 

- 选择 sync.Map 的实现方式需要 golang 1.9^
- v0.2.x 不兼容 v0.1.x  master 分支为开发分支，最新版本

## FAQ

### 选择 sync.Map 实现 

利用 sync.map 达到读取性能相对更高，sync.Map 并不太适应大量写入的缓存操作, 且因为计数使用了 LoadOrStrore 对 key 计数。
sync.Map 在内存空间上并不占优势，约 rwMutex + map 的2倍。

在 4 核以内的机器上锁竞争不明显， 所以 RwMutex map 在性能上更占优势，但是当 cpu 核数 往上时， 锁竞争变大， sync.Map 的优势就体现出来了。
性能测试 引用 https://medium.com/@deckarep/the-new-kid-in-town-gos-sync-map-de24a6bf7c2c

### 选择 rwMutex + map 实现

读写都比较均衡，同时内存占用比 sync.Map 约小1倍，如果读读取要求不强烈。建议选择此实现方式。



