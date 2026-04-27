import logging
import os
from statistics import mean, median, stdev

import requests
from flask import Flask, jsonify, request

app = Flask(__name__)

logging.basicConfig(
    level=logging.INFO,
    format="[analyzer] %(asctime)s %(levelname)s %(message)s",
)
logger = logging.getLogger(__name__)

COLLECTOR_URL = os.getenv("COLLECTOR_URL", "http://localhost:8081")


def fetch_metrics():
    try:
        resp = requests.get(f"{COLLECTOR_URL}/metrics", timeout=5)
        resp.raise_for_status()
        return resp.json()
    except requests.RequestException as e:
        logger.error("Failed to fetch metrics from collector: %s", e)
        return None


def compute_stats(values):
    if not values:
        return {"count": 0, "min": 0, "max": 0, "mean": 0, "median": 0, "stdev": 0}

    result = {
        "count": len(values),
        "min": min(values),
        "max": max(values),
        "mean": round(mean(values), 4),
        "median": round(median(values), 4),
    }
    result["stdev"] = round(stdev(values), 4) if len(values) > 1 else 0
    return result


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "analyzer"})


@app.route("/analyze")
def analyze():
    metrics = fetch_metrics()
    if metrics is None:
        return jsonify({"error": "failed to fetch metrics from collector"}), 502

    grouped = {}
    for m in metrics:
        name = m.get("name", "unknown")
        grouped.setdefault(name, []).append(m.get("value", 0))

    analysis = {}
    for name, values in grouped.items():
        analysis[name] = compute_stats(values)

    logger.info("Analyzed %d metric groups", len(analysis))
    return jsonify({"analysis": analysis, "total_metrics": len(metrics)})


@app.route("/analyze/query", methods=["POST"])
def analyze_query():
    body = request.get_json()
    if not body or "values" not in body:
        return jsonify({"error": "request body must contain 'values' array"}), 400

    values = body["values"]
    if not isinstance(values, list):
        return jsonify({"error": "'values' must be an array of numbers"}), 400

    try:
        float_values = [float(v) for v in values]
    except (TypeError, ValueError):
        return jsonify({"error": "all values must be numeric"}), 400

    stats = compute_stats(float_values)
    logger.info("Computed stats for %d values", len(float_values))
    return jsonify({"stats": stats})


def create_app():
    return app


if __name__ == "__main__":
    port = int(os.getenv("ANALYZER_PORT", "8082"))
    logger.info("Starting analyzer service on port %d", port)
    app.run(host="0.0.0.0", port=port)
