import unittest
import uuid

from fastapi.testclient import TestClient

from database import (
    AgentEventRecord,
    AgentRecord,
    ConfigTemplateRecord,
    FileRecord,
    SessionLocal,
    localnow,
)
from main import app


class AgentsAPITests(unittest.TestCase):
    def setUp(self):
        self.test_id = uuid.uuid4().hex
        self.agent_id = f"agent-{self.test_id}"
        self.agent_ids = {self.agent_id}
        self.hostname = f"host-{self.test_id[:8]}"
        suffix = int(self.test_id[:2], 16)
        self.ip = f"10.0.{suffix // 16}.{(suffix % 16) + 1}"
        self.client = TestClient(app)

    def tearDown(self):
        self.client.close()
        db = SessionLocal()
        try:
            db.query(AgentEventRecord).filter(AgentEventRecord.agent_id.in_(self.agent_ids)).delete(
                synchronize_session=False
            )
            db.query(AgentRecord).filter(AgentRecord.agent_id.in_(self.agent_ids)).delete(
                synchronize_session=False
            )
            db.query(ConfigTemplateRecord).filter(
                ConfigTemplateRecord.name.in_(
                    [f"cfg-{self.test_id}", f"cfg-list-{self.test_id}"]
                )
            ).delete(synchronize_session=False)
            db.query(FileRecord).filter(
                FileRecord.filename == f"node-push-exporter-9.9.{int(self.test_id[:2], 16)}-linux-amd64.tar.gz"
            ).delete(synchronize_session=False)
            db.query(AgentRecord).filter(AgentRecord.hostname == self.hostname, AgentRecord.ip == self.ip).delete(
                synchronize_session=False
            )
            db.commit()
        finally:
            db.close()

    def test_register_creates_agent_and_returns_intervals(self):
        response = self.client.post(
            "/api/agents/register",
            json=self.register_payload(),
        )

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["heartbeat_interval_seconds"], 30)
        self.assertEqual(response.json()["offline_timeout_seconds"], 90)

        db = SessionLocal()
        try:
            row = db.query(AgentRecord).filter(AgentRecord.agent_id == self.agent_id).one()
            self.assertEqual(row.hostname, self.hostname)
            self.assertEqual(row.status, "online")
            self.assertEqual(row.version, "1.2.3")
        finally:
            db.close()

    def test_register_updates_existing_agent(self):
        first = self.client.post(
            "/api/agents/register",
            json=self.register_payload(),
        )
        self.assertEqual(first.status_code, 200)

        updated_payload = self.register_payload()
        updated_payload["hostname"] = "host-02"
        updated_payload["version"] = "1.2.4"
        second = self.client.post(
            "/api/agents/register",
            json=updated_payload,
        )

        self.assertEqual(second.status_code, 200)

        db = SessionLocal()
        try:
            rows = db.query(AgentRecord).filter(AgentRecord.agent_id == self.agent_id).all()
            self.assertEqual(len(rows), 1)
            self.assertEqual(rows[0].hostname, "host-02")
            self.assertEqual(rows[0].version, "1.2.4")
        finally:
            db.close()

    def test_register_reuses_existing_node_when_agent_id_changes(self):
        first = self.client.post(
            "/api/agents/register",
            json=self.register_payload(),
        )
        self.assertEqual(first.status_code, 200)

        db = SessionLocal()
        try:
            record = db.query(AgentRecord).filter(AgentRecord.agent_id == self.agent_id).one()
            record.last_seen_at = None
            record.status = "offline"
            db.commit()
        finally:
            db.close()

        replacement_agent_id = f"agent-{uuid.uuid4().hex}"
        self.agent_ids.add(replacement_agent_id)
        replacement_payload = self.register_payload(agent_id=replacement_agent_id)
        replacement_payload["version"] = "1.2.4"

        second = self.client.post(
            "/api/agents/register",
            json=replacement_payload,
        )

        listing = self.client.get("/api/agents")

        self.assertEqual(second.status_code, 200)
        self.assertEqual(listing.status_code, 200)
        self.assertEqual(len(listing.json()["agents"]), 1)
        self.assertEqual(listing.json()["agents"][0]["agent_id"], replacement_agent_id)
        self.assertEqual(listing.json()["agents"][0]["status"], "online")

        db = SessionLocal()
        try:
            rows = (
                db.query(AgentRecord)
                .filter(AgentRecord.hostname == self.hostname, AgentRecord.ip == self.ip)
                .all()
            )
            self.assertEqual(len(rows), 1)
            self.assertEqual(rows[0].agent_id, replacement_agent_id)
            self.assertEqual(rows[0].version, "1.2.4")
        finally:
            db.close()

    def test_heartbeat_updates_agent_status_and_list(self):
        registered = self.client.post(
            "/api/agents/register",
            json=self.register_payload(),
        )
        self.assertEqual(registered.status_code, 200)

        heartbeat = self.client.post(
            "/api/agents/heartbeat",
            json={
                "agent_id": self.agent_id,
                "status": "degraded",
                "last_error": "push failed",
                "last_push_at": "2026-03-27T12:00:00Z",
                "last_push_success_at": "2026-03-27T11:59:00Z",
                "last_push_error_at": "2026-03-27T12:00:00Z",
                "push_fail_count": 2,
                "node_exporter_up": False,
            },
        )

        listing = self.client.get("/api/agents")

        self.assertEqual(heartbeat.status_code, 200)
        self.assertEqual(listing.status_code, 200)
        self.assertEqual(len(listing.json()["agents"]), 1)
        agent = listing.json()["agents"][0]
        self.assertEqual(agent["status"], "degraded")
        self.assertEqual(agent["last_error"], "push failed")
        self.assertFalse(agent["node_exporter_up"])
        self.assertTrue(agent["online"])

    def test_list_versions_returns_binary_versions_and_config_templates(self):
        db = SessionLocal()
        try:
            db.add(
                FileRecord(
                    filename=f"node-push-exporter-9.9.{int(self.test_id[:2], 16)}-linux-amd64.tar.gz",
                    program="node-push-exporter",
                    version=f"9.9.{int(self.test_id[:2], 16)}",
                    os="linux",
                    arch="amd64",
                    file_path=f"/tmp/node-push-exporter-{self.test_id}.tar.gz",
                    file_size=1024,
                    uploaded_at=localnow(),
                )
            )
            db.add(
                ConfigTemplateRecord(
                    name=f"cfg-list-{self.test_id}",
                    version=f"cfg-{self.test_id[:8]}",
                    content="pushgateway.url=http://pushgateway:9091\npushgateway.job=node",
                    notes="for tests",
                )
            )
            db.commit()
        finally:
            db.close()

    def test_get_config_template_returns_content(self):
        template_name = f"cfg-{self.test_id}"
        db = SessionLocal()
        try:
            template = ConfigTemplateRecord(
                name=template_name,
                version="2026.04.09",
                content="pushgateway.url=http://example.com\npushgateway.job=node",
                notes="test template",
            )
            db.add(template)
            db.commit()
            db.refresh(template)
            template_id = template.id
        finally:
            db.close()

        response = self.client.get(f"/api/config-templates/{template_id}")

        self.assertEqual(response.status_code, 200)
        payload = response.json()
        self.assertEqual(payload["id"], template_id)
        self.assertEqual(payload["name"], template_name)
        self.assertEqual(payload["version"], "2026.04.09")
        self.assertEqual(payload["content"], "pushgateway.url=http://example.com\npushgateway.job=node")

        response = self.client.get("/api/versions")

        self.assertEqual(response.status_code, 200)
        payload = response.json()
        binary_versions = [
            item for item in payload["binary_versions"] if item["version"] == f"9.9.{int(self.test_id[:2], 16)}"
        ]
        config_templates = [
            item for item in payload["config_templates"] if item["name"] == f"cfg-list-{self.test_id}"
        ]
        self.assertEqual(len(binary_versions), 1)
        self.assertEqual(binary_versions[0]["program"], "node-push-exporter")
        self.assertEqual(len(config_templates), 1)

    def register_payload(self, agent_id=None):
        return {
            "agent_id": agent_id or self.agent_id,
            "hostname": self.hostname,
            "version": "1.2.3",
            "os": "linux",
            "arch": "amd64",
            "ip": self.ip,
            "pushgateway_url": "http://pushgateway:9091",
            "push_interval_seconds": 30,
            "node_exporter_port": 9100,
            "node_exporter_metrics_url": "http://127.0.0.1:9100/metrics",
            "current_config_version": "default",
            "started_at": "2026-03-27T11:58:00Z",
        }

if __name__ == "__main__":
    unittest.main()
