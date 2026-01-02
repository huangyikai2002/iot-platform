# iot-platform
IoT-Platform（Go + MQTT + MySQL + Redis + Python AI） 基于 EMQX 的物联网数据链路项目：设备通过 MQTT 上报温度/压力等遥测数据，Go 服务订阅主题后完成设备 Token 校验、数据入库 MySQL，并用 Redis 维护设备在线状态与心跳；同时调用 Python AI 服务进行异常检测，AI 响应慢/不可用时支持超时与降级，保证主链路稳定。项目提供 HTTP API 用于设备注册、查询在线设备及获取设备最新数据，并配套 Python Simulator 支持多设备批量模拟与压测
