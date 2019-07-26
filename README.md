# gocache

[![Build Status](https://travis-ci.org/huzhongqing/gocache.svg?branch=master)](https://travis-ci.org/huzhongqing/gocache)
[![codecov](https://codecov.io/gh/huzhongqing/gocache/branch/master/graph/badge.svg)](https://codecov.io/gh/huzhongqing/gocache)
[![Go Report Card](https://goreportcard.com/badge/github.com/huzhongqing/gocache)](https://goreportcard.com/report/github.com/huzhongqing/gocache)

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
- v0.2.x 不兼容 v0.1.x  master 分支为最新版本

## FAQ

### 为什么选择 sync.Map 

利用 sync.map 达到读取性能相对更高，sync.Map 并不太适应大量写入的缓存操作, 且因为计数使用了 LoadOrStrore 对 key 计数。
sync.Map 在空间上并不占优势。如果存在频繁写入建议使用 RwMutex map  github.com/patrickmn/go-cache

在 4 核 机器上 锁竞争不明显， 所以 RwMutex map 在性能上更占优势，但是当 cpu 核数 往上时， 锁竞争变大， sync.Map 的优势就体现出来了。
性能测试 引用 https://medium.com/@deckarep/the-new-kid-in-town-gos-sync-map-de24a6bf7c2c



