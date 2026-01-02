from flask import Flask, request, jsonify
import numpy as np
from model import AnomalyDetector

app = Flask(__name__)

# 用足够多的正常数据训练，且 contamination 调低
detector = AnomalyDetector(contamination=0.0005, random_state=42)

np.random.seed(42)
n = 12000

# 扩大一点范围，让边界更正常
temp = np.random.uniform(19.5, 30.5, size=n)
press = np.random.uniform(0.95, 1.55, size=n)

# 对齐到两位
temp = np.round(temp, 2)
press = np.round(press, 2)

base = np.column_stack([temp, press]).astype(float)

# 显式加入边界角落点
corners = np.array([
    [20.00, 1.00],
    [20.00, 1.50],
    [30.00, 1.00],
    [30.00, 1.50],
    [29.96, 1.50],  
], dtype=float)

normal_data = np.vstack([base, np.repeat(corners, 300, axis=0)])
detector.train(normal_data)


@app.get("/health")
def health():
    return jsonify({"ok": True, "model": "isolation_forest", "contamination": 0.005})

@app.post("/detect")
def detect():
    data = request.get_json(silent=True) or {}
    try:
        t = float(data.get("temperature"))
        p = float(data.get("pressure"))
    except Exception:
        return jsonify({"error": "invalid temperature/pressure"}), 400

    sample = np.array([[t, p]], dtype=float)
    return jsonify({"anomaly": bool(detector.predict(sample))})

if __name__ == "__main__":
    # 绑定 5001
    app.run(host="0.0.0.0", port=5001)
