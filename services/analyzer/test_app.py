import json
from unittest.mock import patch, MagicMock

import pytest

import requests

from app import app, compute_stats, fetch_metrics


@pytest.fixture
def client():
    app.config["TESTING"] = True
    with app.test_client() as client:
        yield client


def test_health(client):
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["status"] == "ok"
    assert data["service"] == "analyzer"


def test_compute_stats_empty():
    stats = compute_stats([])
    assert stats["count"] == 0
    assert stats["mean"] == 0


def test_compute_stats_single():
    stats = compute_stats([42.0])
    assert stats["count"] == 1
    assert stats["min"] == 42.0
    assert stats["max"] == 42.0
    assert stats["mean"] == 42.0
    assert stats["stdev"] == 0


def test_compute_stats_multiple():
    stats = compute_stats([10, 20, 30, 40, 50])
    assert stats["count"] == 5
    assert stats["min"] == 10
    assert stats["max"] == 50
    assert stats["mean"] == 30.0
    assert stats["median"] == 30


def test_analyze_query_success(client):
    resp = client.post(
        "/analyze/query",
        data=json.dumps({"values": [10, 20, 30]}),
        content_type="application/json",
    )
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["stats"]["count"] == 3
    assert data["stats"]["mean"] == 20.0


def test_analyze_query_missing_values(client):
    resp = client.post(
        "/analyze/query",
        data=json.dumps({"data": [1, 2]}),
        content_type="application/json",
    )
    assert resp.status_code == 400


def test_analyze_query_invalid_values(client):
    resp = client.post(
        "/analyze/query",
        data=json.dumps({"values": "not_a_list"}),
        content_type="application/json",
    )
    assert resp.status_code == 400


def test_analyze_query_non_numeric(client):
    resp = client.post(
        "/analyze/query",
        data=json.dumps({"values": ["a", "b"]}),
        content_type="application/json",
    )
    assert resp.status_code == 400


@patch("app.fetch_metrics")
def test_analyze_success(mock_fetch, client):
    mock_fetch.return_value = [
        {"name": "cpu", "value": 50},
        {"name": "cpu", "value": 70},
        {"name": "mem", "value": 80},
    ]
    resp = client.get("/analyze")
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["total_metrics"] == 3
    assert "cpu" in data["analysis"]
    assert "mem" in data["analysis"]
    assert data["analysis"]["cpu"]["count"] == 2


@patch("app.fetch_metrics")
def test_analyze_collector_down(mock_fetch, client):
    mock_fetch.return_value = None
    resp = client.get("/analyze")
    assert resp.status_code == 502


@patch("app.requests.get")
def test_fetch_metrics_success(mock_get):
    mock_resp = MagicMock()
    mock_resp.json.return_value = [{"name": "test", "value": 1}]
    mock_resp.raise_for_status.return_value = None
    mock_get.return_value = mock_resp
    result = fetch_metrics()
    assert len(result) == 1


@patch("app.requests.get")
def test_fetch_metrics_failure(mock_get):
    mock_get.side_effect = requests.RequestException("connection refused")
    result = fetch_metrics()
    assert result is None
