# batch_start.py  (单进程多设备)
import threading
import time
import json
import random
import requests
import paho.mqtt.client as mqtt

MQTT_BROKER = "127.0.0.1"
MQTT_PORT = 1883
API_REGISTER_URL = "http://127.0.0.1:8080/devices/register"
NUM_DEVICES = 50
SEND_INTERVAL = 2

def register():
    r = requests.post(API_REGISTER_URL, timeout=3)
    r.raise_for_status()
    d = r.json()
    return d["device_id"], d["token"]

def run_device(idx: int):
    device_id, token = register()
    topic = f"devices/{device_id}/data"

    client = mqtt.Client(client_id=device_id)

    def on_connect(c, u, f, rc, properties=None):
        if rc == 0:
            print(f"[{idx}] connected {device_id}")
        else:
            print(f"[{idx}] connect failed rc={rc}")

    client.on_connect = on_connect
    client.connect(MQTT_BROKER, MQTT_PORT, 60)
    client.loop_start()

    try:
        while True:
            payload = {
                "device_id": device_id,
                "token": token,
                "temperature": round(random.uniform(20, 30), 2),
                "pressure": round(random.uniform(1.0, 1.5), 2),
                "timestamp": int(time.time())
            }
            client.publish(topic, json.dumps(payload), qos=1)
            # print(f"[{idx}] {device_id} -> {payload}")
            time.sleep(SEND_INTERVAL)
    finally:
        client.loop_stop()
        client.disconnect()

def main():
    threads = []
    for i in range(NUM_DEVICES):
        t = threading.Thread(target=run_device, args=(i,), daemon=True)
        t.start()
        threads.append(t)
        time.sleep(0.05)  # 避免瞬间打爆注册 API / MQTT broker

    print(f"✅ running {NUM_DEVICES} devices in ONE process")
    while True:
        time.sleep(10)

if __name__ == "__main__":
    main()

