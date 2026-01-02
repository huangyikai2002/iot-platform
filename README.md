# IoT 异常检测与在线监测平台 IoT-Platform（Go + MQTT + MySQL + Redis + Python AI）

一个可本地一键运行的物联网数据链路示例：设备通过 MQTT 上报遥测数据，Go 服务消费消息并入库 MySQL、用 Redis 维护在线状态，同时调用 Python AI 服务做异常检测；提供 HTTP API 便于注册设备、查询在线设备和设备最新数据，并提供 Python Simulator 进行多设备模拟与压测。

## 功能概览
- MQTT：设备上报数据（EMQX Broker）
- Go 消费端：订阅 Topic → 校验/解析 → 入库 MySQL → 写 Redis 在线状态 → 调用 AI 检测（超时/降级）
- HTTP API：注册设备、查询在线设备、查询最新数据
- Python AI：Flask 服务，异常检测接口
- Python Simulator：单设备/批量模拟上报

## 技术栈
- Go：HTTP API / MQTT Consumer / Worker Pool / 熔断降级
- EMQX：MQTT Broker
- MySQL：数据存储
- Redis：在线状态（TTL / ZSET）
- Python：Flask +（依赖：numpy/sklearn等）

## 端口
- Go HTTP：`http://127.0.0.1:8080`
- AI 服务：`http://127.0.0.1:5001`
- MQTT：`127.0.0.1:1883`
- EMQX Dashboard：`http://127.0.0.1:18083`

---

## 快速开始（本地运行）
### 1）启动基础设施（EMQX/MySQL/Redis）
```bash
docker compose up -d --pull never
docker ps
```
### 2）启动 Python AI
```bash
cd python-ai
python -m venv venv
# Windows 推荐不 activate，直接用 venv python：
./venv/Scripts/python.exe -m pip install -r requirements.txt
./venv/Scripts/python.exe app.py

### 验证： curl http://127.0.0.1:5001/health
```

### 3）启动 Go 服务
```bash
cd cmd/server
go run .

### 验证： curl http://127.0.0.1:8080/health
```

### 4）启动设备模拟器（单设备/批量）
```bash
cd python-simulator
python -m venv venv
./venv/Scripts/python.exe -m pip install -r requirements.txt
./venv/Scripts/python.exe simulator.py
# 或批量：
./venv/Scripts/python.exe batch_start.py
```

### API示例
- 注册设备： 
```bash
curl -X POST http://127.0.0.1:8080/devices/register
```
- 在线设备： 
```bash
curl "http://127.0.0.1:8080/devices/online?ttl=10"
```
- 最新数据： 
```bash
curl "http://127.0.0.1:8080/devices/<device_id>/latest"
```