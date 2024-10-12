# Research

## 一、项目介绍
- ##### 分布式搜索引擎，提供集群部署和单节点使用。

- ##### 使用分片map实现倒排索引，同时将数据保存到正排索引中。再次使用时可从正排索引加载到内存中

- ##### 实现一致性哈希算法保证集群高性能

- ##### 副本数据使用Raft共识性算法保证集群高可用

## 二、部署方法

### 1、集群模式

- #### etc/etc.yaml

  ~~~yaml
  # 单节点还是集群 node/cluster
  configType: cluster
  ~~~

- #### node结点上需要填写etcd、索引、注册中心和server配置

   - ~~~yaml
      # etcd配置
      etcd:
        endpoints:
        - 192.168.92.201:2379
        - 192.168.92.201:2372
        dialTimeout: 300
      # 正排索引配置
      forwardindex:
        dbType: 1  #使用数据库类型
        dateDir: 192.168.92.201:6379 #数据库地址
      # 倒排索引配置
      reverseindex:
        indexType: 1 #使用索引结构类型，默认是跳表
        docNumEstimate: 1000 #文档数目预估
      # 服务注册中心配置
      servicehub:
        serviceHubType: proxy
        heartbeatFrequency: 3
      #限流，默认使用令牌桶算法
      limit:
        capacity: # 令牌桶容量
        rate:   # 放入令牌数量 个/s
        tokens: # 初始令牌数量
      #集群模式下node节点配置
      server :
        nodeIp : 192.168.92.201
        port : 5455
      ~~~
   - ##### 启动main.go程序 读取配置文件信息，向etcd注册服务，监听指定端口暴露服务

- #### master节点上需要填写etcd和服务注册中心配置

  - ~~~yaml
    # etcd配置
    etcd:
      endpoints:
      - 192.168.92.201:2379
      - 192.168.92.201:2372
      dialTimeout: 300
    # 服务注册中心配置
    servicehub:
      serviceHubType: proxy
      heartbeatFrequency: 3
    ~~~

  - #### 调用Research.go中的方法进行使用

      ~~~go
      c := etc.GetConfig("etc/etc.yaml")
      rs,_ := NewResearch(c)
      ~~~

### 2、单节点模式

- ### etc/etc.yaml

  ~~~yaml
  # 单节点还是集群 node/cluster
  configType: node
  ~~~

- ### 填写除server和limit外的配置信息，使用Research进行操作

  ~~~yaml
  c := etc.GetConfig("etc/etc.yaml")
  rs,_ := NewResearch(c)
  ~~~

## 三、集群架构

![](./docs/img/Research架构.png)

### 一致性哈希

- 数据通过一致性哈希分布在多个colony结点上，每个colony都有一个服务名

### Colony

- 每个colony结点实际上是一个使用**Raft共识算法的集群**，数据拥有多个副本
- Raft集群达成日志协商后，进行日志提交，即转发请求给node处理

### Raft Client

- 对于事务请求master节点与 raft Client通信，raft Client 转发请求给Leader进行日志协商
- 对于非事务请求master节点直接与node通信
- 监听etcd 内所属raft集群节点信息

### master

- 接收用户请求，事务请求通过负载均衡找到raft Client结点，非事务请求通过负载均衡找到node节点直接通信
- 监听etcd的node节点信息和raft client节点信息

### 配置文件解析

~~~yaml
# 集群模式下node节点配置
server :
  groupId: research1 # 根据该id分配到不同的集群
  nodeIp : 192.168.92.201
  port : 5455
  weight: 0.15 # 服务器节点权重，最大为1

# 集群模式下master节点配置
master:
  groupId: research1,research2 # 服务名
  virtualNodeNum: 100,200  # 每个服务下虚拟节点数量
~~~

## 四、组件扩展

### 1、负载均衡

- 接口

  ~~~go
  type LoadBalancer interface {
  	Take([]*EndPoint) *EndPoint
  }
  ~~~

- 提供方案

  - 基于轮询的方式（默认）
  - 基于随机的方式
  - 基于权重的方式

- 切换方案

  只有集群模式才可进行切换

  ~~~go
  func (r *Research) SetLoadBalancer(lb ServiceHub.LoadBalancer) error
  ~~~

### 2、限流策略

- #### 限流接口

  - ~~~go
      type Limiter interface {
          Allow(ctx context.Context) error // 等待请求达到通过条件
      }
    ~~~

  - ##### 实现该接口即可进行限流策略转换，Allow方法要求在一定时间内等待请求通过

- #### 客户端使用

    - ~~~go
      func NewResearch(c *etc.Config, limiter ServiceHub.Limiter) (*Research, error)
      ~~~

    - ##### 调用NewResearch传入自定义的结构体

- #### 默认配置

    - ~~~yaml
      #限流，默认使用令牌桶算法
      limit:
        capacity: # 令牌桶容量
        rate:   # 放入令牌数量 个/s
        tokens: # 初始令牌数量
      ~~~

    - ##### 令牌桶算法相关配置

- #### 只有集群模式下注册中心使用代理模式才有限流

### 3、倒排索引结构
### 4、正排索引数据库