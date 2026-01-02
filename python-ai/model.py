import numpy as np
from sklearn.ensemble import IsolationForest

class AnomalyDetector:
    def __init__(self, contamination=0.005, random_state=42):
        self.model = IsolationForest(
            n_estimators=400,
            contamination=contamination,
            random_state=random_state
        )
        self.trained = False

    def train(self, data: np.ndarray):
        self.model.fit(data)
        self.trained = True

    def predict(self, sample: np.ndarray) -> bool:
        # sample: shape (1, 2)
        if not self.trained:
            return False
        pred = self.model.predict(sample)[0]  # 1 normal, -1 anomaly
        return pred == -1
