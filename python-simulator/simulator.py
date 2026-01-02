import time
import json
import random
import sys
import requests
import paho.mqtt.client as mqtt

MQTT_BROKER = "127.0.0.1"
MQTT_PORT = 1883
SEND_INTERVAL = 2
TOPIC_TEMPLATE = "devices/{}/data"
API_REGISTER_URL = "http://127.0.0.1:8080/devices/register"

# 第 60 秒开始发送异常数据
ANOMALY_AFTER_SEC = 60

def register():
    resp = requests.post(API_REGISTER_URL, timeout=3)
    resp.raise_for_status()
    data = resp.json()
    return data["device_id"], data["token"]

def main():
    # batch_start会传入index
    idx = sys.argv[1] if len(sys.argv) > 1 else "0"

    try:
        device_id, token = register()
    except Exception as e:
        print(f"[{idx}] ❌ register failed:", e)
        return

    print(f"[{idx}] ✅ registered: {device_id}")

    # paho-mqtt v2 
    client = mqtt.Client(client_id=device_id)

    def on_connect(client, userdata, flags, reason_code, properties=None):
        if reason_code == 0:
            print(f"[{idx}] 🌐 MQTT connected")
        else:
            print(f"[{idx}] ❌ MQTT connect failed:", reason_code)

    client.on_connect = on_connect
    client.connect(MQTT_BROKER, MQTT_PORT, 60)
    client.loop_start()

    start = time.time()
    try:
        while True:
            elapsed = time.time() - start
            if elapsed >= ANOMALY_AFTER_SEC:
                temperature = round(random.uniform(80, 120), 2)
                pressure = round(random.uniform(3.0, 6.0), 2)
            else:
                temperature = round(random.uniform(20, 30), 2)
                pressure = round(random.uniform(1.0, 1.5), 2)

            payload = {
                "device_id": device_id,
                "token": token,
                "temperature": temperature,
                "pressure": pressure,
                "timestamp": int(time.time())
            }
            topic = TOPIC_TEMPLATE.format(device_id)
            client.publish(topic, json.dumps(payload), qos=1)
            print(f"[{idx}] 📤", payload)
            time.sleep(SEND_INTERVAL)
    except KeyboardInterrupt:
        print(f"[{idx}] ⏹ stopped")
    finally:
        client.loop_stop()
        client.disconnect()

if __name__ == "__main__":
    main()
